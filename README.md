# Go SoftPack Analytics

This package provides a simple server that listens on a port for simple analytical information, to which it adds simple metadata and prints.

## Usage

The server takes the following arguments:

|   Argument   |   Default   |  Description                      |
|--------------|-------------|-----------------------------------|
| -p           | 1234        | The TCP port to listen on.        |
| -d           |             | DB file to write to.              |
| -t           |             | TSV file to import into database. |

NB: TSV file import is to import flatfile database created with earlier version of this program.

## Output

The generated file will be an SQLite Database with the following tables:

events:

|   Column   |   Type   |   Description                                                                             |
|------------|------------------------------------------------------------------------------------------------------|
| username   | String   | The user the ran the executable.Human                                                     |
| command    | String   | The path of the executable that was passed to the analytics server.                       |
| ip         | String   | The IP Address on which the executable was ran.                                           |
| time       | Integer  | The Unix timestamp (Seconds since 1970-01-01 00:00:00 UTC) when the command was executed. |

modules:

|   Column   |   Type   |   Description                                         |
|------------|----------|-------------------------------------------------------|
| module     | String   | Module that executable is determined to be a part of. |
| count      | Integer  | Number of times that this module has been used.       |
| firstuse   | Integer  | Unix timestamp of the earliest use of the module.     |
| lastuse    | Integer  | Unix timestamp of the latest used of the module.      |

usermodules:

|   Column   |   Type   |   Description                                                  |
|------------|----------|----------------------------------------------------------------|
| module     | String   | Module that executable is determined to be a part of.          |
| username   | String   | User than ran the executable.                                  |
| count      | Integer  | Number of times that this module has been used by this user.   |
| firstuse   | Integer  | Unix timestamp of the earliest use of the module by this user. |
| lastuse    | Integer  | Unix timestamp of the latest used of the module by this user.  |


## Sending Data

This server can recieve information in a very simple format which consists of a username and an executable path, seperated by a null byte.

An example usage is below:

```bash
{
        (echo -e "$USER\0$0" > /dev/tcp/server-domain/1234 2> /dev/null) &
} 2> /dev/null
```

…where server-domain is the domain name that the analytics server is running on and 1234 is the port it is listening on.
