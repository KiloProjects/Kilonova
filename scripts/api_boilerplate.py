import argparse
import requests
from dataclasses import dataclass
from typing import Any, Literal, Optional
from urllib.parse import urljoin

@dataclass
class APIResponse:
    status: Literal['success', 'error']
    data: Any

@dataclass
class UserBrief:
    id: int
    name: str
    admin: bool
    proposer: bool
    bio: Optional[str] = None

@dataclass(kw_only=True)
class UserFull(UserBrief):
    email: str
    verified_email: str
    preferred_language: str
    created_at: str
    generated: str

class Client():
    def __init__(self, base: str):
        self.token = "guest"
        self.base = base
    
    def get(self, path: str, params: Optional[dict]=None) -> APIResponse:
        res = requests.get(urljoin(self.base, path), params=params, headers={"Authorization": self.token}).json()
        val = res.json()
        return APIResponse(val['status'], val['data'])

    def post(self, path: str, data: Optional[dict]=None) -> APIResponse:
        res = requests.post(urljoin(self.base, path), data=data, headers={"Authorization": self.token})
        val = res.json()
        return APIResponse(val['status'], val['data'])

    def login(self, uname: str, pwd: str) -> None:
        res = self.post("/api/auth/login", {"username": uname, "password": pwd})
        if res.status == "error":
            raise Exception(res.data)
        self.token = res.data

    def generate_user(self, uname: str) -> tuple[str, UserFull]:
        res = self.post("/api/user/generateUser", {"username": uname})
        if res.status == "error":
            raise Exception(res.data)
        return res.data["password"], UserFull(**res.data["user"])

    def register_user_in_contest(self, uname: str, contest_id: int) -> None:
        res = self.post(f"/api/contest/{contest_id}/forceRegister", {'name': uname})
        if res.status == "error":
            raise Exception(res.data)
    
    # TODO: Leaderboard stuffs, JSON object for anonymization/deanonymization, do not try to create existing users


if __name__ == '__main__':
    parser = argparse.ArgumentParser(prog="kn_scripter", description="Scripting for Kilonova")
    parser.add_argument('-u', '--username', required="true")
    parser.add_argument('-p', '--password', required="true")
    args = parser.parse_args()
    
    cl = Client("http://localhost:8070/")
    cl.login(args.username, args.password)
    
    print(cl.generate_user("sufar_de_palpitatii"))
    
    # TODO