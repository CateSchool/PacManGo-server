## Program structure

Contrary to Go official guidelines, I'm including the entire workspace under version control, 
rather than `src/*` directory (officially called *repositories*). This was the simplest way to 
include other files such as [websockets.html](/websockets.html).

## Running the program

From the project root directory, run the following commands:

```
. .env
go run server
```
