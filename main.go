package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reusee/lgo"
	"github.com/vmihailenco/msgpack"
)

var (
	pt = fmt.Printf
)

const luaCodes = `
	lgi = require 'lgi'
	Gtk = lgi.require('Gtk', '3.0')
	Gdk = lgi.Gdk
	Pango = lgi.Pango
	Vte = lgi.Vte

	--local css_provider = Gtk.CssProvider()
	--css_provider:load_from_data([[
	--vte-terminal {
	--    padding: 10px 20px;
	--}
	--]])
	--Gtk.StyleContext.add_provider_for_screen(
	--	Gdk.Screen.get_default(),
	--	css_provider,
	--	Gtk.STYLE_PROVIDER_PRIORITY_USER)

	local window = Gtk.Window{type = Gtk.WindowType.TOPLEVEL}
	window:set_decorated(false)
	window:set_title('vte-neovim')
	window:set_role('VteNeovim')
	window.on_delete_event = function()
		return true
	end

	local term = Vte.Terminal.new()
	term:set_cursor_shape(Vte.CursorShape.BLOCK)
	term:set_cursor_blink_mode(Vte.CursorBlinkMode.OFF)
	term:set_font(Pango.FontDescription.from_string('xos4 Terminus 13'))
	term:set_color_cursor(Gdk.RGBA().parse('#fcaf17'))
	term:set_color_cursor_foreground(Gdk.RGBA().parse('black'))
	term:set_scrollback_lines(-1)
	term:set_scroll_on_output(false)
	term:set_scroll_on_keystroke(true)
	term:set_rewrap_on_resize(true)
	term:set_encoding('UTF-8')
	term:set_allow_bold(true)
	term:set_allow_hyperlink(true)
	term:set_mouse_autohide(true)
	term:set_cjk_ambiguous_width(2)

	term:spawn_sync(
		Vte.PtyFlags.DEFAULT,
		'.',
		{
			'/usr/bin/nvim',
			[[+ :call serverstart('::1:]] .. rpc_port() .. [[')]],
			''
		}, 
		{},
		0,
		function() end,
		nil,
		nil,
		0,
		nil,
		nil,
		nil
	)
	term.on_child_exited = function()
		Sys_exit()
	end
	term.on_button_press_event = function(widget, ev)
		if ev.button == 3 then
			term:copy_clipboard()
		end
	end

	window:add(term)

	window:show_all()

	Gtk.main()

`

func main() {
	lua := lgo.NewLua()

	rpcPort := rand.Intn(40000) + 10000

	go startNvimClient(rpcPort)

	lua.RegisterFunctions(map[string]interface{}{
		"Sys_exit": func() {
			os.Exit(0)
		},
		"rpc_port": func() string {
			return fmt.Sprintf("%d", rpcPort)
		},
	})

	lua.RunString(luaCodes)
}

func startNvimClient(rpcPort int) {

dial:
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", rpcPort))
	if err != nil {
		time.Sleep(time.Millisecond * 200)
		goto dial
	}
	defer conn.Close()

	var msgID int64
	type Call struct {
		Done   chan struct{}
		Return interface{}
		Error  interface{}
	}
	var calls sync.Map
	syncCall := func(
		method string,
		args ...interface{},
	) (
		ret interface{},
		err error,
	) {
		if args == nil {
			args = make([]interface{}, 0)
		}
		id := atomic.AddInt64(&msgID, 1)
		done := make(chan struct{})
		call := &Call{
			Done: done,
		}
		calls.Store(id, call)
		defer calls.Delete(id)
		if err := msgpack.NewEncoder(conn).Encode([]interface{}{
			0,
			id,
			method,
			args,
		}); err != nil {
			panic(err)
		}
		select {
		case <-done:
			if call.Error != nil {
				err = fmt.Errorf("error: %v", call.Error)
				return
			}
			ret = call.Return
			return
		case <-time.After(time.Second * 5):
			err = fmt.Errorf("%s call timeout", method)
			return
		}
		return
	}

	go func() {
		for {
			var data []interface{}
			if err := msgpack.NewDecoder(conn).Decode(&data); err != nil {
				return
			}
			pt("RECEIVE: %v\n", data)
			switch data[0].(int8) {

			case 1:
				id := reflect.ValueOf(data[1]).Int()
				if v, ok := calls.Load(id); ok {
					call := v.(*Call)
					call.Error = data[2]
					call.Return = data[3]
					close(call.Done)
				}

			case 2:

			}
		}
	}()

	if _, err := syncCall("nvim_ui_attach", 800, 600, map[string]interface{}{
		"rgb": true,
	}); err != nil {
		panic(err)
	}

}

func init() {
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}
