package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/reusee/lgo"
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
	n, err := nvim.Dial(fmt.Sprintf("localhost:%d", rpcPort))
	if err != nil {
		time.Sleep(time.Millisecond * 100)
		goto dial
	}
	pt("connected %v\n", n)

}

func init() {
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}
