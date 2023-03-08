# jnote 

## Introduce
Only for easily create/search/modify notes (in a single JSON file).
![screenshot](jnote2.gif)

## Support OS
- Mac
- Linux

## Installation
```bash
$ git clone https://github.com/Databingo/jnote
$ cd jnote && CGO_ENABLED=1 go install
```
## Usage
```bash
jnote note.json
```
Enter jnote, then press 'a' to create a new note opened by Vim, write your note then type ":wq" back to jnote, all your notes will be saved in note.json automatically.

## Other Keybinding
Similiar as the original repository `tson` except 'a' means add new node to root node.

| Key    | description                    |
|--------|--------------------------------|
| a      | add new note                   |
| D      | delete a note                  |
| / or f | search                         |
| j      | move down                      |
| k      | move up                        |
| Enter  | edit note by VIM               |
| q      | quit jnote                     |

Actually you can use jnote as a JSON editor, for more usage information you can check the original repository which is no longer be developed by the author.(https://github.com/skanehira/tson)

## Suggestions
1. Add ```alias jn="jnote note.json"``` to .bashrc for easy use.
2. Use [transcrypt](https://github.com/elasticdog/transcrypt) to encrypt JSON then push to Github's private repository for notes' persistence.

## Todo
1. Set current node when use G or gg.
2. ~~Order records according created time descending when build nodes tree.(Done)~~
3. Tidy code.
4. Auto save vim content to file per 5 seconds & escape loop when exit from edition (immediately).
5. Set current node as up simbling node after delete node in notes (and in search result?).
6. Fix one more click needness after back from edition.
7. Deal with syscall.SIGWINCH so could use on windows?

## About improvement
Since the original repository was last committed at 2019.11.07, many dependences changed. I updated them and add a Text area for the visitation of node's content.



