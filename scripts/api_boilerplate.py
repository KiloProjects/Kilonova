from typing import Any, Literal, Optional
import dataclasses
from urllib.parse import urljoin
import pandas as pd
import requests
import argparse
import io
import json


@dataclasses.dataclass
class APIResponse:
    status: Literal["success", "error"]
    data: Any


@dataclasses.dataclass
class UserBrief:
    id: int
    name: str
    admin: bool
    proposer: bool
    bio: Optional[str] = None


@dataclasses.dataclass(kw_only=True)
class UserFull(UserBrief):
    email: str
    verified_email: str
    preferred_language: str
    created_at: str
    generated: str


@dataclasses.dataclass
class UserBundle:
    contest_id: int
    anon_prefix: str
    to_generate: Optional[list[str]] = None
    created_users: Optional[dict[str, tuple[str, str]]] = None


class Client:
    def __init__(self, base: str):
        self.token = "guest"
        self.base = base

    def get(self, path: str, params: Optional[dict] = None) -> APIResponse:
        res = requests.get(
            urljoin(self.base, path),
            params=params,
            headers={"Authorization": self.token},
        )
        val = res.json()
        return APIResponse(val["status"], val["data"])

    def post(self, path: str, data: Optional[dict] = None) -> APIResponse:
        res = requests.post(urljoin(self.base, path), data=data, headers={"Authorization": self.token})
        val = res.json()
        return APIResponse(val["status"], val["data"])

    def postform(self, path: str, files: Optional[dict] = None) -> APIResponse:
        res = requests.post(
            urljoin(self.base, path),
            files=files,
            headers={"Authorization": self.token},
        )
        val = res.json()
        return APIResponse(val["status"], val["data"])

    def login(self, uname: str, pwd: str) -> None:
        res = self.post("/api/auth/login", {"username": uname, "password": pwd})
        if res.status == "error":
            raise Exception(res.data)
        self.token = res.data
        return None

    def merge_tags(self, into: int, org: list[int]) -> None:
        for tg in org:
            res = self.post("/api/tags/merge", {"to_keep": into, "to_replace": tg})
            if res.status != "success":
                print(f"Couldn't merge tag {tg} in {into}: {res.data}")

    def check_username_exists(self, uname: str) -> bool:
        res = self.get("/api/user/getByName", {"name": uname})
        return res.status == "success"

    def generate_user(self, uname: str) -> tuple[str, UserFull]:
        res = self.post("/api/user/generateUser", {"username": uname})
        if res.status == "error":
            raise Exception(res.data)
        return res.data["password"], UserFull(**res.data["user"])

    def register_user_in_contest(self, uname: str, contest_id: int) -> None:
        res = self.post(f"/api/contest/{contest_id}/forceRegister", {"name": uname})
        if res.status == "error":
            raise Exception(res.data)
        return None

    def leaderboard_csv(self, contest_id: int) -> pd.DataFrame:
        res = requests.get(
            urljoin(self.base, f"/assets/contest/{contest_id}/leaderboard.csv"),
            cookies={"kn-sessionid": self.token},
        )
        content = res.content.decode("utf-8")
        if res.status_code != 200:
            raise Exception(content)
        return pd.read_csv(io.StringIO(content))

    def deanonymized_leaderboard(self, bundle: UserBundle) -> pd.DataFrame:
        df = self.leaderboard_csv(bundle.contest_id)
        df["username"] = df["username"].map(lambda x: bundle.created_users.get(x, [x, "no_pwd"])[0] if bundle.created_users is not None else x)
        return df

    def generate_users(self, bundle: UserBundle) -> UserBundle:
        id = 100
        if not bundle.created_users:
            bundle.created_users = {}
        for person in bundle.to_generate:
            while self.check_username_exists(f"{bundle.anon_prefix}{id}"):
                id += 1
            pwd, user = self.generate_user(f"{bundle.anon_prefix}{id}")
            self.register_user_in_contest(user.name, bundle.contest_id)
            bundle.created_users[user.name] = (person, pwd)
        bundle.to_generate = []
        return bundle

    def load_bundle(self, path: str) -> UserBundle:
        with open(path, "r") as f:
            b = UserBundle(**json.load(f))
        b = self.generate_users(b)
        self.save_bundle(path, b)
        return b

    def format_user_info(self, bundle: UserBundle) -> str:
        out = ""
        for anon, user in bundle.created_users.items():
            out += f"""
Concurent: {user[0]}
Username: {anon}
Parolă: {user[1]}
-----------------------
"""
        return out

    def save_bundle(self, path: str, bundle: UserBundle) -> None:
        with open(path, "w") as f:
            json.dump(dataclasses.asdict(bundle), f)
        return None

    def upload_test_archive(self, problem_id: int, path: str) -> None:
        file = ""
        with open(path, "rb") as f:
            file = f.read()
        print("Uploading test archive...")
        res = self.postform(
            f"/api/problem/{problem_id}/update/processTestArchive",
            files={"testArchive": file},
        )
        print(res)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(prog="kn_scripter", description="Scripting for Kilonova")
    parser.add_argument("-u", "--username", required=True)
    parser.add_argument("-p", "--password", required=True)
    parser.add_argument("-bp", "--bundle_path")
    args = parser.parse_args()
    print("Logging in")
    cl = Client("http://localhost:8070/")
    cl.login(args.username, args.password)

    print("Logged in")
    # val = cl.leaderboard_csv(1)
    # print(b)

    # cl.register_user_in_contest("GJ_6_100", 3)

    # cl.upload_test_archive(6, "./teste_influent.zip")

    b = cl.load_bundle(args.bundle_path)

    print("Loaded bundle")
    ld = cl.deanonymized_leaderboard(b)
    ld.to_csv("./deanonimizat_C.csv")
    print(ld)
    # info = cl.format_user_info(b)
    # print(info)
    # ld.to_csv(args.bundle_path + ".csv")
