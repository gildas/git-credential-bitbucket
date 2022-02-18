# git-credential-bitbucket

This is a git credential store for bitbucket's OAUTH.  

It can be used with git remotes when neither the ssh protocol nor the basic authentication are available. Typically, the bitbucket account/repo has been configure to use OAUTH.

## Pre-requisites

You will need to create a bitbucket consumer to be able to use this credential store. See https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html.  
Make sure you create a **private** consumer so you can use the `client_credentials` OAUTH grant type (the Callback URL can be a dummy).  
As for the permissions, you need to give **admin** permissions to `repositories` and **write** to `Pull requests`.

Once created, collect the **key** and the **secret** from the consumer (click on the consumer name on the OAuth page)

## Installation

### Binaries

Download the binary for your platform from the [releases](https://github.com/gildas/git-credential-bitbucket/releases) page.

Unzip it and copy the executable in a folder that belongs to your `PATH`.

### Source

Get the source by cloning this repository and run:  
```console
make install
```

By default the binary is installed in `/usr/local/bin`. You can choose another folder as long as it belongs to your `PATH` environment variable.

## Usage

### Adding credentials

You can add credentials by running the git credential tools:  
```console
git credential-bitbucket store <<EOM
protocol=https
host=bitbucket.org
username=xxx
clientid=yyy
secret=zzz
```

Where:  
- `xxx` is your username on bitbucket.org,
- `yyy` is the **key** you collected in the Pre-requisites,
- `zzz` is the **secret** you collected in the Pre-requisites.

### Using the credential store

In the git repository clone on your machine, make sure the remote repository is something like:
```console
git remote add bitbucket https://xxx@bitbucket.org/path/to/repo.git
```

Where `xxx` is your username on bitbucket.org (or at least the same username you used to store the credentials).

Then, add the credential helper:

```console
git config credential.helper bitbucket
```

Now, the next time you do a `git pull`, `git push` on that remote repository, the credential helper will manage the OAUTH token for you.

### Removing credentials

If for any reason, you don't need the credentials anymore, you can simply run:  
```console
git credential-bitbucket erase <<EOM
protocol=https
host=bitbucket.org
username=xxx
```

Where `xxx` is your username on bitbucket.org (or at least the same username you used to store the credentials).

### Advanced usage

#### Store location

By default, the credential store is stored in `$XDG_DATA_HOME/git-credential-bitbucket` (we follow the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html))

The folder used can be changed with:  
- the environment variable `STORE_LOCATION`,
- the command line argument `--store-location`.

Example:  
```console
git config credential.helper "bitbucket --store-location /path/to/location"
```

**Note:** Make sure the store is readable only by yourself as it will contain sensible information.

#### Bitbucket Workspaces

If your account is used in various Bitbucket workspaces you can store per-workspace credentials:  
```console
git credential-bitbucket --workspace MyTeam store <<EOM
protocol=https
host=bitbucket.org
username=xxx
clientid=yyy
secret=zzz
```

And you would configure the helper in git as follows:  
```console
git config credential.helper "bitbucket --workspace MyTeam"
```

#### Token Renewal

By default, the credential store will try to renew the bitbucket's OAUTH token 10 minutes before it expires.

This can be changed with:  
- the environment variable `RENEW_BEFORE`,
- the command line argument `--renew`.

Example, to renew 20 minutes before the token expire:  
```console
git config credential.helper "bitbucket --renew 20m"
```

#### Logging

If, for any reason, you need to analyze what happens when the credential store is used, you can add logging:  
```console
git config credential.helper "bitbucket --log /path/to/log/credentials.log"
```
The log is stored using the [gildas/go-logger](https://github.com/gildas/go-logger) format and can be (pretty) read with [bunyan](https://github.com/trentm/node-bunyan).

**Note:** the logs can contain ids and passwords.
