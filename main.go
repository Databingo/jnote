package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/gofrs/uuid"
	"github.com/rivo/tview"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	moveNext int = iota + 1
	movePre
)

var (
	ErrEmptyJSON = errors.New("empty json")
)

func run() int {
	os.Stdin = os.NewFile(uintptr(syscall.Stderr), "/dev/tty")

	//var err error
	if len(os.Args) <= 1 {
		log.Fatal("please give json file name")
	}

	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal("Error when get json file: ", err)
	}

	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		log.Fatal("Error when unmarshal json file: ", err)
	}

	// log.Printf("%s", payload)
	//	log.Printf("%s", i)

	if err := New().Run(i); err != nil {
		//if err := New(); err != nil {
		return 1
	}
	return 0

}

type Gui struct {
	Tree  *Tree
	Navi  *Navi
	App   *tview.Application
	Pages *tview.Pages
	Text  *tview.TextView
}

func New() *Gui {
	g := &Gui{
		Tree:  NewTree(),
		Navi:  NewNavi(),
		App:   tview.NewApplication(),
		Pages: tview.NewPages(),
		Text:  tview.NewTextView(),
	}
	return g
}

func (g *Gui) Run(i interface{}) error {
	g.Tree.UpdateView(g, i)
	g.Tree.SetKeybindings(g)
	g.Navi.UpdateView()
	g.Navi.SetKeybindings(g)
	g.Text.SetTextAlign(tview.AlignLeft).SetText("Text").SetBorder(true).SetTitle("text content")
	//g.Text.SetKeybindings(g)
	//g.Text.SetDoneFunc(func(key tcell.Key) {
	//	if key == tcell.KeyEnter {
	//		log.Println("child.GetText()")
	//		g.EditWithEditor()
	//	}
	//})
	grid := tview.NewGrid().
		SetColumns(-1, -1).
		AddItem(g.Tree, 0, 0, 1, 1, 0, 0, true).
		AddItem(g.Text, 0, 1, 1, 1, 0, 0, true)

	g.Pages.AddAndSwitchToPage("main", grid, true)

	if err := g.App.SetRoot(g.Pages, true).Run(); err != nil {
		return err
	}
	return nil
}

func (g *Gui) Search() {
	pageName := "search"
	if g.Pages.HasPage(pageName) {
		g.Pages.ShowPage(pageName)
	} else {
		input := tview.NewInputField()
		input.SetBorder(true).SetTitle("search").SetTitleAlign(tview.AlignLeft)
		input.SetChangedFunc(func(text string) {
			root := *g.Tree.OriginRoot
			g.Tree.SetRoot(&root)
			if text != "" {
				root := g.Tree.GetRoot()
				root.SetChildren(g.walk(root.GetChildren(), text))
			}
		})
		input.SetLabel("word").SetLabelWidth(5).SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				g.Pages.HidePage(pageName)
			}
		})
		g.Pages.AddAndSwitchToPage(pageName, g.Modal(input, 0, 3), true).ShowPage("main")
	}
}

func (g *Gui) walk(nodes []*tview.TreeNode, text string) []*tview.TreeNode {
	var newNodes []*tview.TreeNode
	for _, child := range nodes {
		if strings.Index(strings.ToLower(child.GetText()), strings.ToLower(text)) != -1 {
			newNodes = append(newNodes, child)
		} else {
			newNodes = append(newNodes, g.walk(child.GetChildren(), text)...)

		}
	}
	return newNodes
}

func (g *Gui) get_parent_node_by_id(p_node *tview.TreeNode, id string) []*tview.TreeNode {

	var parent_nodes []*tview.TreeNode
	nodes := p_node.GetChildren()
	for _, child := range nodes {
		if child.GetReference().(Reference).ID == id {
			parent_nodes = append(parent_nodes, p_node)
			break
		} else {
			parent_nodes = append(parent_nodes, g.get_parent_node_by_id(child, id)...)
		}
	}
	return parent_nodes

}

func (g *Gui) AddNode() {
	labels := []string{"json"}
	g.Form(labels, "add", "add new node", "add_new_node", 7, func(values map[string]string) error {
		j := values[labels[0]]
		if j == "" {
			log.Println("empty json")
			return ErrEmptyJSON
		}
		buf := bytes.NewBufferString(j)
		i, err := UnMarshalJSON(buf)
		if err != nil {
			return err
		}

		newNode := NewRootTreeNode(i)
		newNode.SetChildren(g.Tree.AddNode(i))
		g.Tree.GetCurrentNode().AddChild(newNode)
		g.Tree.OriginRoot = g.Tree.GetRoot()

		return nil

	})

}

func (g *Gui) AddNode_2() {
	// recover if in search results treeview
	root := *g.Tree.OriginRoot
	g.Tree.SetRoot(&root)

	// construct one note json as object{}
	dt := fmt.Sprintf(time.Now().Format("2006.01.01-15:04:05"))
	j := fmt.Sprintf(`{"%v":""}`, dt)

	buf := bytes.NewBufferString(j)
	i, err := UnMarshalJSON(buf)
	if err != nil {
		log.Println(err)
	}

	newNode := NewRootTreeNode(i)
	newNode.SetChildren(g.Tree.AddNode(i))
	newKeyNode := newNode.GetChildren()[0]
	newTextNode := newNode.GetChildren()[0].GetChildren()[0]

	if txt := g.Tree.GetRoot().GetText(); txt == "{array" {
		g.Tree.GetRoot().AddChild(newNode)
		g.Tree.OriginRoot = g.Tree.GetRoot()
		g.Tree.SetCurrentNode(newTextNode)

	} else {

		g.Tree.GetRoot().AddChild(newKeyNode)
		g.Tree.OriginRoot = g.Tree.GetRoot()
		g.Tree.SetCurrentNode(newTextNode)
	}
	//--------------

	t := g.Tree
	text := t.GetCurrentNode().GetText()

	g.Text.SetText(text)

	g.EditWithEditor(t)

}

func (g *Gui) Form(fieldLabel []string, doneLabel, title, pageName string, height int, doneFunc func(values map[string]string) error) {
	if g.Pages.HasPage(pageName) {
		g.Pages.ShowPage(pageName)
		return
	}

	form := tview.NewForm()
	for _, label := range fieldLabel {
		form.AddInputField(label, "", 0, nil, nil)
	}

	form.AddButton(doneLabel, func() {
		values := make(map[string]string)
		for _, label := range fieldLabel {
			item := form.GetFormItemByLabel(label)
			switch item.(type) {
			case *tview.InputField:
				input, ok := item.(*tview.InputField)
				if ok {
					values[label] = os.ExpandEnv(input.GetText())
				}
			}
		}

		if err := doneFunc(values); err != nil {
			//	g.Message(err.Error(), pageName, func() {})
			log.Println(err.Error(), "error")
			return
		}

		g.Pages.RemovePage(pageName)
	}).
		AddButton("cancel", func() {
			g.Pages.RemovePage(pageName)
		})

	form.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	g.Pages.AddAndSwitchToPage(pageName, g.Modal(form, 0, height), true).ShowPage("main")

}

var json_str string

func (g *Gui) Save_json_2() {
	root := *g.Tree.OriginRoot
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	//enc.
	enc.Encode(g.MakeJSON(&root))

	ioutil.WriteFile(os.Args[1], buf.Bytes(), 0666)
	log.Println("saved\r\n")
}

func (g *Gui) DeleteNode() {
	t := g.Tree
	node_this := t.GetCurrentNode()
	id := node_this.GetReference().(Reference).ID

	root := g.Tree.OriginRoot
	root.Walk(func(node, parent *tview.TreeNode) bool {
		nid := node.GetReference().(Reference).ID
		if nid == id && parent != nil {
			parent.RemoveChild(node)
			// delele in search result
			nodes_of_search := g.Tree.GetRoot().GetChildren()
			for _, node := range nodes_of_search {
				nid := node.GetReference().(Reference).ID
				if nid == id {
					g.Tree.GetRoot().RemoveChild(node)
				}

			}

			g.Text.SetText("")
			g.Save_json_2()

			return true
		}
			return true
	})
}

func (g *Gui) Save_json() {
	root_s := *g.Tree.OriginRoot
	r := root_s.GetText()
	p_s := ""

	switch r {
	case "{object}":
		json_str = "{}"
	case "{array}":
		json_str = "[]"
		p_s = p_s + ".yesiamamarkforarray"
	case "{value}":
		json_str, _ = sjson.SetRaw(root_s.GetReference().(Reference).OnlySimpleValue, "", "")
	}

	// ValueType: null boolean number string array object
	// AsKey: Fasle True

	children := root_s.GetChildren()
	g.walk_json(children, p_s)

	// save edited json to file
	fs, _ := os.OpenFile("save.json", os.O_CREATE, 0666)
	if err := ioutil.WriteFile(fs.Name(), []byte(json_str), 0666); err != nil {
		log.Println(fmt.Sprintf("can't create json file: %s", err))
	}
	defer fs.Close()

}

func (g *Gui) walk_json(nodes []*tview.TreeNode, ps string) {

	for _, child := range nodes {

		var psr string
		psr = strings.Clone(ps) // psr for deal in this loop time  in circle ; ps for pass to next children nodes

		json_type := child.GetReference().(Reference).JSONType

		text := child.GetText()

		var idx int64

		text = strings.Replace(text, ":", "placeholderforclean", -1)
		// check numberic key for escape . in path_string
		if json_type != Value {
			if _, err := strconv.ParseFloat(text, 64); err == nil {
				text = ":" + text // for sjson use number as key}
				//		log.Println("^^^^", text, "\r\n")
			}
			text = strings.Replace(text, ".", "\\.", -1) // escape .
		}

		has_key_part := child.GetReference().(Reference).KeyPart // node has key part
		var array_len_check_str string
		var inte int
		var flt float64
		// check array
		path_array := strings.Split(ps, ".")
		if path_array[len(path_array)-1] == "yesiamamarkforarray" {
			if len(path_array) == 2 { //for only one array as whole json ["", "yesiamamarkforarray"]
				array_len_check_str = "#"
				idx = gjson.Get(json_str, array_len_check_str).Int()
				psr = strconv.FormatInt(idx, 10)
			} else { // get array length then put index(for insert position) into path_string
				array_len_check_str = strings.Join(path_array[:len(path_array)-1], ".") + ".#"
				array_len_check_str = strings.Replace(array_len_check_str, ":", "", -1) // for gjson no need:
				idx = gjson.Get(json_str, array_len_check_str).Int()
				text = strings.Replace(text, "placeholderforclean", ":", -1)
				// log.Println(idx)
				psr = strings.Join(append(path_array[:len(path_array)-1], strconv.FormatInt(idx, 10)), ".")
			}

			log.Println("original_json:", json_str, "--in-array--len_check_str:", array_len_check_str, "get_idx:", idx, "construct_psr:", psr, "\r\n")
		}

		switch json_type {
		case Object:
			if has_key_part {
				if psr != "" {
					psr = psr + "." + text
				} else {
					psr = text
				}
			}
		case Array:
			if has_key_part {
				psr = psr + "." + text + ".yesiamamarkforarray"
			} else {
				psr = psr + ".yesiamamarkforarray"
			}
		case Key:
			if psr != "" {
				psr = psr + "." + text
			} else {
				psr = text
			}
		case Value:
			value_type := child.GetReference().(Reference).ValueType
			switch value_type {
			case String:
				text = strings.Replace(child.GetText(), "\\.", ".", -1) //  return \. to .
				json_str, _ = sjson.Set(json_str, psr, text)
			case Int:
				inte, _ = strconv.Atoi(text)
				json_str, _ = sjson.Set(json_str, psr, inte)
			case Float:
				flt, _ = strconv.ParseFloat(text, 64)
				json_str, _ = sjson.Set(json_str, psr, flt)
			default:
				json_str, _ = sjson.Set(json_str, psr, text)

			}

			//		json_str, _ = sjson.Set(json_str, psr, text)
			log.Println("*path_str:", psr, "*set_text:", text, "*get json_str:", json_str, "\r\n")
			log.Println("*set_text:", text, "*result_json_str:", json_str, "\r\n")
		}

		//log.Println("<<<ps:", ps,  "\r\n")
		g.walk_json(child.GetChildren(), psr)

	}
}

func (g *Gui) Modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

//	func (g *Gui) ModalT(p tview.Primitive, width, height int) tview.Primitive {
//		return tview.NewGrid().
//			SetColumns(0, width, 0).
//			SetRows(0, height, 0).
//			AddItem(p, 1, 2, 1, 1, 0, 0, true)
//	}
func (g *Gui) MakeJSON(node *tview.TreeNode) interface{} {
	ref := node.GetReference().(Reference)
	children := node.GetChildren()

	switch ref.JSONType {
	case Object:
		i := make(map[string]interface{})
		for _, n := range children {
			i[n.GetText()] = g.MakeJSON(n)
		}
		return i
	case Array:
		var i []interface{}
		for _, n := range children {
			i = append(i, g.MakeJSON(n))
		}
		return i
	case Key:
		if len(node.GetChildren()) == 0 {
			return ""
		}
		v := node.GetChildren()[0]
		if v.GetReference().(Reference).JSONType == Value {
			return g.parseValue(v)
		}
		return map[string]interface{}{
			node.GetText(): g.MakeJSON(v),
		}
	}
	return g.parseValue(node)

}

func (g *Gui) parseValue(node *tview.TreeNode) interface{} {
	v := node.GetText()
	ref := node.GetReference().(Reference)

	switch ref.ValueType {
	case Int:
		i, _ := strconv.Atoi(v)
		return i
	case Float:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	case Boolean:
		b, _ := strconv.ParseBool(v)
		return b
	case Null:
		return nil
	}
	return v

}

func (g *Gui) Input(text, label string, width int, doneFunc func(text string)) {
	//input := tview.NewInputField().SetText(text)
	//input.SetBorder(true)
	//input.SetLabel(label).SetLabelWidth(width).SetDoneFunc(func(key tcell.Key) {
	//		if key == tcell.KeyEnter {
	//			doneFunc(input.GetText())
	//			g.Pages.RemovePage("input")
	//		}
	//})
	//g.Pages.AddAndSwitchToPage("input", g.Modal(input, 0, 3), true).ShowPage("main")
}

func (g *Gui) EditWithEditor(t *Tree) {

	g.App.Suspend(func() {
		text := t.GetCurrentNode().GetText()

		f, err := ioutil.TempFile("", "tson")
		ioutil.WriteFile(f.Name(), []byte(text), 0644)
		//f, err := os.Open("note.json")
		f.Close()

		editor := os.Getenv("EDITOR")
		if editor == "" {
			log.Println("$EDITOR is empty")
			return
		}
		var args []string
		//var sch = fmt.Sprintf("/%s", text)
		if editor == "vim" {
			args = append(args, []string{"-c", "set ft=json", f.Name()}...)
			//args = append(args, []string{"-c", sch, f.Name()}...)
		}
		cmd := exec.Command(editor, args...)
		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Println("open $EDITOR failed")
			return
		}
		defer func() {
			if err := ptmx.Close(); err != nil {
				log.Println("can't close pty: %s", err)
			}
		}()
		//---------------------
		// Handle pty size.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					log.Printf("can't resizing pty: %s", err)
				}
			}
		}()
		ch <- syscall.SIGWINCH // Initial resize.

		// Set stdin in raw mode.
		oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			log.Println(fmt.Sprintf("can't make terminal raw mode: %s", err))
			//	g.Message(err.Error(), "main", func() {})
			return
		}
		defer func() {
			if err := terminal.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				log.Printf("can't restore terminal: %s", err)
			}
		}()

		// Copy stdin to the pty and the pty to stdout.
		go io.Copy(ptmx, os.Stdin)
		io.Copy(os.Stdout, ptmx)
		//---------------------

		//	if err != nil {
		//		log.Println(fmt.Sprintf("can't open file: %s", err))
		//		//	g.Message(err.Error(), "main", func() {})
		//		return
		//	}
		//	defer f.Close()

		//	i, err := UnMarshalJSON(f)
		//	if err != nil {
		//		log.Println(fmt.Sprintf("can't read from file: %s", err))
		//		//	g.Message(err.Error(), "main", func() {})
		//		return
		//	}
		b_edited, err := ioutil.ReadFile(f.Name())
		texted := string(b_edited[:])
		texted = strings.TrimSuffix(texted, "\n")

		tr := t.GetCurrentNode()
		tr.SetText(texted)

		ref := tr.GetReference().(Reference)
		ref.ValueType = parseValueType(texted)
		tr.SetReference(ref)

		// just for show change
		g.Text.SetText(texted)

		//g.Save_json()
		g.Save_json_2()
		os.RemoveAll(f.Name())
		//	g.Tree.UpdateView(g, i)
		//---------------------
	})
}

var NaviPageName = "navi_panel"
var RedColor = `[red::b]%s[white]: %s`
var (
	moveDown    = fmt.Sprintf(RedColor, "j", "     move down")
	moveUp      = fmt.Sprintf(RedColor, "k", "     move up")
	defaultNavi = strings.Join([]string{moveDown, moveUp}, "\n")
)

var (
	hideNode    = fmt.Sprintf(RedColor, "h", " hide children nodes")
	searchNodes = fmt.Sprintf(RedColor, "/ or f", " search nodes")
	showText    = fmt.Sprintf(RedColor, "t", " show text")
	treeNavi    = strings.Join([]string{hideNode, searchNodes, showText}, "\n")
)

type Navi struct {
	*tview.TextView
}

func NewNavi() *Navi {
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetBorder(true).SetTitle("help").SetTitleAlign(tview.AlignLeft)
	navi := &Navi{TextView: view}
	return navi
}

func (n *Navi) UpdateView() {
	navi := strings.Join([]string{defaultNavi, "", treeNavi}, "\n")
	n.SetText(navi)
}

func (n *Navi) SetKeybindings(g *Gui) {
	n.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			g.Pages.HidePage(NaviPageName)
		}
		return event
	})
}

func (t *Tree) SetKeybindings(g *Gui) {
	t.SetSelectedFunc(func(node *tview.TreeNode) {
		text := node.GetText()
		if text == "{object}" || text == "{array}" || text == "{value}" {
			return
		}
		labelWidth := 5
		g.Input(text, "text", labelWidth, func(text string) {
			ref := node.GetReference().(Reference)
			ref.ValueType = parseValueType(text)
			if ref.ValueType == String {
				text = strings.Trim(text, `"`)
			}
			node.SetText(text).SetReference(ref)
		})
	})
	t.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'h':
			t.GetCurrentNode().SetExpanded(false)
		case 'H':
			t.CollapseValues(t.GetRoot())
		case 'L':
			t.GetRoot().ExpandAll()
		case 'l':
			t.GetCurrentNode().SetExpanded(true)
		case '/', 'f':
			g.Search()
		case 'q':
			g.App.Stop()
		case 'j':
			t.ShowTextDown(g)
		case 'k':
			t.ShowTextUp(g)
		case 'a':
			g.AddNode_2()//add {"datetime":""} to root & open value by vim
			g.Save_json_2()
		case 'A':
			g.AddNode()// serious add json format objects
			g.Save_json_2()
		case 'D':
			g.DeleteNode()
                case ' ':
		        current := t.GetCurrentNode()
			current.SetExpanded(!current.IsExpanded())
                case 'e':
			g.EditWithEditor(t)
			//t.GetCurrentNode().ClearChildren()
			//node_this := t.GetCurrentNode()
			//id := node_this.GetReference().(Reference).ID
			////parent_nodes := g.get_parent_node_by_id(g.Tree.GetRoot(), id)
			//parent_nodes := g.get_parent_node_by_id(g.Tree.OriginRoot, id)
			//log.Println(parent_nodes)
			//log.Println(len(parent_nodes))
			//if len(parent_nodes) > 0 {
			//	// delete in originroot note
			//	parent_node := parent_nodes[0]
			//	//t.Move(-1).GetCurrentNode()
			//	t.Move(-1)
			//	parent_node.RemoveChild(node_this)
			//	// delele in search result
			//	nodes_of_search := g.Tree.GetRoot().GetChildren()
			//	for _, node := range nodes_of_search {
			//		nid := node.GetReference().(Reference).ID
			//		if nid == id {
			//			g.Tree.GetRoot().RemoveChild(node)
			//		}
			//	}
			//	g.Text.SetText("")
			//	g.Save_json_2()
			//}

		}
		switch event.Key() {
		case tcell.KeyCtrlJ:
			t.moveNode(moveNext)
		case tcell.KeyCtrlK:
			t.moveNode(movePre)
	//	case tcell.KeyEnter:
	//		g.EditWithEditor(t)
		}
		return event
	})
}

func (t *Tree) ShowTextDown(g *Gui) {
	//current := t.GetCurrentNode()
	current := t.Move(1).GetCurrentNode()
	text := current.GetText()
	t.Move(-1).GetCurrentNode()
	g.Text.SetText(text)
}

func (t *Tree) ShowTextUp(g *Gui) {
	//current := t.GetCurrentNode()
	current := t.Move(-1).GetCurrentNode()
	text := current.GetText()
	t.Move(1).GetCurrentNode()
	g.Text.SetText(text)
}

func (t *Tree) moveNode(movement int) {
	current := t.GetCurrentNode()
	t.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
		if parent != nil {
			children := parent.GetChildren()
			for i, n := range children {
				if n.GetReference().(Reference).ID == current.GetReference().(Reference).ID {
					if movement == moveNext {
						if i < len(children)-1 {
							t.SetCurrentNode(children[i+1])
							return false
						}
					} else if movement == movePre {
						if i > 0 {
							t.SetCurrentNode(children[i-1])
							return false
						}
					}
				}
			}
		}
		return true
	})
}

type Tree struct {
	*tview.TreeView
	OriginRoot *tview.TreeNode
}

func NewTree() *Tree {
	t := &Tree{
		TreeView: tview.NewTreeView(),
	}
	t.SetBorder(true).SetTitle("json tree").SetTitleAlign(tview.AlignLeft)
	return t
}

func (t *Tree) UpdateView(g *Gui, i interface{}) {
	go func() {
		g.App.QueueUpdateDraw(func() {

			root := NewRootTreeNode(i)
			root.SetChildren(t.AddNode(i))
			t.SetRoot(root).SetCurrentNode(root)

			originRoot := *root
			t.OriginRoot = &originRoot
		})
	}()
}

func NewRootTreeNode(i interface{}) *tview.TreeNode {
	r := reflect.ValueOf(i)
	var root *tview.TreeNode
	id := uuid.Must(uuid.NewV4()).String()
	switch r.Kind() {
	case reflect.Map:
		root = tview.NewTreeNode("{object}").SetReference(Reference{ID: id, JSONType: Object, KeyPart: false})
	case reflect.Slice:
		root = tview.NewTreeNode("{array}").SetReference(Reference{ID: id, JSONType: Array, KeyPart: false})
		// default: // other json data types such as string, number, null, bool as {value}
		//        root = tview.NewTreeNode("{value}").SetReference(Reference{JSONType: Key, OnlySimpleValue: fmt.Sprintf("%v", i)})
	case reflect.String:
		root = tview.NewTreeNode("{value}").SetReference(Reference{ID: id, JSONType: Key, OnlySimpleValue: fmt.Sprintf("%v", i)})
	case reflect.Float64:
		root = tview.NewTreeNode("{value}").SetReference(Reference{ID: id, JSONType: Key, OnlySimpleValue: fmt.Sprintf("%v", i)})
	case reflect.Bool:
		root = tview.NewTreeNode("{value}").SetReference(Reference{ID: id, JSONType: Key, OnlySimpleValue: fmt.Sprintf("%v", i)})
	//case nil:
	default: // other json data null as {value}
		root = tview.NewTreeNode("{value}").SetReference(Reference{ID: id, JSONType: Key, OnlySimpleValue: "null"})
		//root = tview.NewTreeNode("{value}").SetReference(Reference{JSONType: Key, OnlySimpleValue: fmt.Sprintf("%v", i)})

	}
	return root
}

func (t *Tree) AddNode(node interface{}) []*tview.TreeNode {
	var nodes []*tview.TreeNode

	switch node := node.(type) {
	// deal with object
	case map[string]interface{}:
		for k, v := range node {
			newNode := t.NewNodeWithLiteral(k).
				SetColor(tcell.ColorMediumSlateBlue). // key show color blue?
				SetChildren(t.AddNode(v))             // add children to per Node's SetChilren()

			r := reflect.ValueOf(v)
			id := uuid.Must(uuid.NewV4()).String()
			if r.Kind() == reflect.Map {
				newNode.SetReference(Reference{ID: id, JSONType: Object, KeyPart: true})
				//	 newNode.SetReference(Reference{ID: id, JSONType: Key})
			} else if r.Kind() == reflect.Slice {
				newNode.SetReference(Reference{ID: id, JSONType: Array, KeyPart: true})
				//		 newNode.SetReference(Reference{ID: id, JSONType: Key})
			} else {
				newNode.SetReference(Reference{ID: id, JSONType: Key, KeyPart: true}) // if next node is simple value(null bool number string) this node is the value's key
			}
			nodes = append(nodes, newNode)
		}
		// deal with array
	case []interface{}:
		for _, v := range node {
			id := uuid.Must(uuid.NewV4()).String()
			switch v.(type) {
			// object
			case map[string]interface{}:
				objectNode := tview.NewTreeNode("{object}").
					SetChildren(t.AddNode(v)).SetReference(Reference{ID: id, JSONType: Object, KeyPart: false})
				nodes = append(nodes, objectNode)
				// array
			case []interface{}:
				arrayNode := tview.NewTreeNode("{array}").
					SetChildren(t.AddNode(v)).SetReference(Reference{ID: id, JSONType: Array, KeyPart: false})
				nodes = append(nodes, arrayNode)
				// simple value -> next node default
			default:
				nodes = append(nodes, t.AddNode(v)...)
			}
		}
		// deal with simple value
	default:
		ref := reflect.ValueOf(node)
		var valueType ValueType
		switch ref.Kind() {
		case reflect.Int:
			valueType = Int
		case reflect.Float64:
			valueType = Float
		case reflect.Bool:
			valueType = Boolean
		default:
			if node == nil {
				valueType = Null
			} else {
				valueType = String
			}
		}

		id := uuid.Must(uuid.NewV4()).String()
		nodes = append(nodes, t.NewNodeWithLiteral(node).
			SetReference(Reference{ID: id, JSONType: Value, ValueType: valueType}))
	}
	return nodes
}

func (t *Tree) NewNodeWithLiteral(i interface{}) *tview.TreeNode {
	if i == nil {
		return tview.NewTreeNode("null")
	}
	return tview.NewTreeNode(fmt.Sprintf("%v", i))
}

func (t *Tree) CollapseValues(node *tview.TreeNode) {
	node.Walk(func(node, parent *tview.TreeNode) bool {
		ref := node.GetReference().(Reference)
		if ref.JSONType == Value {
			pRef := parent.GetReference().(Reference)
			t := pRef.JSONType
			if t == Key || t == Array {
				parent.SetExpanded(false)
			}
		}
		return true
	})
}

type JSONType int

const (
	Root JSONType = iota + 1
	Object
	Array
	Key
	Value
)

var jsonTypeMap = map[JSONType]string{
	Object: "object",
	Array:  "array",
	Key:    "key",
	Value:  "value",
}

func (t JSONType) String() string {
	return jsonTypeMap[t]
}

type ValueType int

const (
	Int ValueType = iota + 1
	String
	Float
	Boolean
	Null
)

var valueTypeMap = map[ValueType]string{
	Int:     "int",
	String:  "string",
	Float:   "float",
	Boolean: "boolean",
	Null:    "null",
}

func (v ValueType) String() string {
	return valueTypeMap[v]
}

type Reference struct {
	ID              string
	JSONType        JSONType
	ValueType       ValueType
	OnlySimpleValue string
	KeyPart         bool
}

func parseValueType(text string) ValueType {
	if strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
		//	 log.Println("..string")
		return String
	} else if "null" == text {
		//	 log.Println("..null")
		return Null
	} else if text == "fasle" || text == "true" {
		//	 log.Println("..fasle or true")
		return Boolean
	} else if _, err := strconv.ParseFloat(text, 64); err == nil {
		//	 log.Println("..flaot")
		return Float
	} else if _, err := strconv.Atoi(text); err == nil {
		//	 log.Println("..int")
		return Int
	}
	//	 log.Println(".String")
	return String
}

func UnMarshalJSON(in io.Reader) (interface{}, error) {
	b, _ := ioutil.ReadAll(in)
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		log.Println(err)
		return nil, err
	}
	//log.Println(string(b))
	return i, nil
}

func main() {
	os.Exit(run())
}
