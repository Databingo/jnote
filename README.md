# jnote
A note terminal application based on a single JSON file.
![screenshot](jnote.gif)

## Support OS
- Mac
- Linux

## Installation
```bash
$ git clone https://github.com/Databingo/jnote
$ cd jnote && go install
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

## Todo
1. Set current node when use G or gg 
2. Order records according created time when build nodes tree.


## About improvement
Since the original repository was last committed at 2019.11.07, many dependences changed. I updated them and add a Text area for the visitation of node's content.



