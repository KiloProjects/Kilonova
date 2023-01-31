import argparse
import requests

class Client():
    def __init__(self, base: str):
        self.token = "guest"
        self.base = base
    
    def get(self, path: str):
        requests.get(path)
        # TODO
        pass

    def post(self, path: str):
        requests.post(path)
        # TODO
        pass

    def login(self, uname: str, pwd: str):
        pass

if __name__ == '__main__':
    parser = argparse.ArgumentParser(prog="kn_scripter", description="Scripting for Kilonova")
    parser.add_argument('-u', '--username', required="true")
    parser.add_argument('-p', '--password', required="true")
    args = parser.parse_args()
    cl = Client("http://localhost:8070/api/")
    cl.login(args.username, args.password)
    # TODO