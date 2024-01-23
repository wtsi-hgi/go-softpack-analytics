# Go SoftPack Analytics

This package provides a simple server that listens on a port for simple analytical information, to which it adds simple metadata and prints.

## Usage

The server takes the following arguments:

|   Argument   |   Default   |  Description               |
|--------------|-------------|----------------------------|
| -p           | 1234        | The TCP port to listen on. |
| -o           | - (stdout)  | The file to print to.      |

## Output

The generated file will be a TSV with the following fields:

|   Field   |   Description                                                       |
|-----------|---------------------------------------------------------------------|
| Time      | Human readable timestamp of the format: YYYY-MM-DD HH:mm:ss         |
| Path      | The path of the executable that was passed to the analytics server. |
| Username  | The user the ran the executable.                                    |
| IP Address| The IP Address on which the executable was ran.                     |

## Sending Data

This server can recieve information in a very simple format which consists of a username and an executable path, seperated by a null byte.

An example usage is below:

```bash
{
        (echo -e "$USER\0$0" > /dev/tcp/server-domain/1234 2> /dev/null) &
} 2> /dev/null
```

…where server-domain is the domain name that the analytics server is running on and 1234 is the port it is listening on.
