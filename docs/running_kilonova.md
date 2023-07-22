# Setting up kilonova: an interactive guide

## by Alex Vasiluta


So, you want to set up a kilonova instance? Maybe it's the future, I'm dead and you want to keep the legacy, or you just want to set up a private instance for a contest or something. Well, this guide right here is just for you! I'll guide you step by step, explaining everything along the way.


## Prerequisites

- An Ubuntu server (preferably 22.04 or later). Something else may work, but I can't make sure your packages match up correctly;
- Some system administration skills;
- A bit of patience.

## Installing the base goods

```bash
$ sudo apt install build-essential mold clang git golang python3-dev python3-pip python-is-python3
$ pip install toml # Required to build translation blobs on instance start
```

Mold is a faster linker that is used by the c/c++ compilers on the platform. You can consult `eval/languages.go` if you want to see for yourself!
I use clang because I figured at one time that it might be a bit faster, and I just stuck with it.

## Setting up the environment

I generally use the master branch of go for the platform, since I try to make use of bleeding edge features wherever possible. [Here](https://raw.githubusercontent.com/AlexVasiluta/dotfiles/master/scripts/setup/golang.sh) is a script that automatically compiles from source for you. Make sure to add `$HOME/src/go/bin` to $PATH if you do this.

## Setting up postgres

Recent versions of PostgreSQL are recommended since the platform makes heavy use of CTEs and only they optimize them properly.

Here's the setup steps for postgres:

```bash
$ sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
$ wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo tee /etc/apt/trusted.gpg.d/psql.asc
$ sudo apt update && sudo apt install -y postgresql-15
```

This is the part where things get a bit clunky for me, we need to generate the database and the user. My namename is usually `alexv`, and I found that setting it identical to your usual user helps make things easier when manually connecting to the DB. You may replace `alexv` in the following snippets with whatever you like.

```bash
$ sudo su - postgres
$ createuser alexv -s
Password: <insert new user password>
$ createdb kilonova -O alexv
$ exit
```

I usually use password authentication to make sure everything's alright, so we need to enable it in `pg_hba.conf`.

`/etc/postgresql/15/main/pg_hba.conf`
```bash
# ....
# "local" is for Unix domain socket connections only
local   all             all                                     md5
# ....
```

At the beginning, it's set to `peer`, but I always modify to `md5`.

A `sudo systemctl restart postgresql@15-main.service` is probably required to actually deploy the changes.

## Actually building the database structure

In `db/psql_schema/` there's a few files with commands that must be executed in their specified order for them to work neatly. I usually open 2 terminals, one running a `psql -d kilonova` connection and one with all the files printed out. I manually copy-paste from one terminal to another. This may not be the best system for running database migrations, but it's good enough.

## Setting up nodejs and yarn

For better or for worse, we need to set up nodejs and yarn before moving forward, since the next step requires them.

```bash
$ curl -fsSL https://deb.nodesource.com/setup_19.x | sudo -E bash - &&\
	sudo apt-get install -y nodejs
$ npm install --global yarn
```

## Building static JS/CSS bundles

We want our instance to have both style, and functionality, right? Well, we need to set up the javascript bundles first

```bash
$ cd ./web/assets/
$ yarn install # set up node_modules
$ yarn prodJS # bundle custom functions
$ yarn prodCSS # build styles
$ yarn vendor # bundle libraries used directly from html
```

## Setting up `isolate`

I created a simple script that automatically sets up the `isolate` sandbox for you in `scripts/init_isolate.sh`. You may need to run `usermod -a -G kn_sandbox "$USER"` a few times then log out and back in for it to truly take effect.

## Disabling cgroups v2

Modern linux systems are a bit of a pain in the arse since they come with cgroups v2 enabled, but isolate only supports cgroups v1 for now. Thus, we need to disable cgroups v2 from grub. If you don't do this step, you may get a cryptic `error status 2` from isolate!

To do this, go to `/etc/default/grub` and add `systemd.unified_cgroup_hierarchy=0` to `GRUB_CMDLINE_LINUX`. After this, run `sudo update-grub` and restart your server.

## Setting up the cron job

Subtests will start piling up and eating your disk space unless you do something about it. It's been a problem for me quite a few times, in which I run out of storage space and everything goes bonkers. I created (using ChatGPT!) a script that sets up a cron job and a delete script that keeps the subtests under a bearable size limit.

Before running `scripts/setup_crontab.sh`, make sure you modify your `DESTINATION` and `CLEANUP_PATH` variables so it matches your environment.

## Sample `config.toml`

```toml
[common]
 log_dir = "/home/alexv/src/kninfo/logs"
 data_dir = "/home/alexv/src/kninfo/data"
 debug = false # true if you want more debug output, usually from the grader.
 host_prefix = "http://localhost:8070" # Note that it must be without an ending back slash!
 port = 8070
 default_language = "ro"
 db_dsn = "sslmode=disable host=/var/run/postgresql dbname=kilonova user=<YOUR USERNAME> password=<YOUR PSQL PASSWORD> application_name=kilonova"
 test_max_mem_kb = 1048576 # 1gb limit for setting up tests
 updates_webhook = "https://discord.com/api/webhooks/<WEBHOOK_ID>/<WEBHOOK_SECRET>" # Webhook for audit log updates

[eval]
 isolatePath = "/usr/local/etc/isolate_bin"
 compilePath = "/tmp/kncompiles"
 num_concurrent = 3
 global_max_mem_kb = 1048576 # 1gb global memory limit

 starting_box = 1 # If there are multiple kilonova instances active on the system, change to a multiple of 100, just to be safe 

[email]
 host = "smtp.fastmail.com:587"
 username = "<SMTP USERNAME>"
 password = "<SMTP PASSWORD>"

[frontend]
 root_problem_list = 1 # The problem list to show on the front page
 ```

## Finishing touches

Right now, sometimes, the grader sometimes stops properly working when kilonova is not run as root. I don't know why that happens, but it sure as hell is not fun. `sudo ./runkn.sh` will start the platform with the root user and most problems should float away.

You should create the first user now. The first user is automatically granted admin+proposer ranks. I'll also write guides about their respective abilities and how to properly have a nice experience as a kilonova proposer/admin.

Creating a simple `sum`-type problem with some basic tests and sending a sample solution should help you make sure the grader is fully and correctly running.

## Final disclaimer

This may not fully work right now. I'm just noting all the honourable mentions from my last attempts at setting up this platform. The next time I'll set it up, I'll revamp this guide further.

Not included in this guide, but you should also set up a firewall like `ufw`, a reverse proxy like `nginx` and something like `fail2ban` for SSH and nginx.

**Don't even think building a docker image for kilonova.** You *will* need to gather a few workarounds to make `isolate` run properly and it's probably too much of a hassle. In a future in which the grader can be decoupled from the rest of the platform, you could make an argument that the part that is not grading submissions could be dockerized, but that's a hassle for the reader attempting this kind of stuff.

