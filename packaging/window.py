import os
import sys

os.environ.setdefault('WEBKIT_DISABLE_DMABUF_RENDERER', '1')

import gi

gi.require_version('Gtk', '3.0')
gi.require_version('Gdk', '3.0')
try:
    gi.require_version('WebKit2', '4.1')
except ValueError:
    gi.require_version('WebKit2', '4.0')
from gi.repository import Gdk, GLib, Gtk, WebKit2

if len(sys.argv) > 1 and sys.argv[1] == '--check':
    sys.exit(0)

url = sys.argv[1]
icon = sys.argv[2] if len(sys.argv) > 2 else ''

GLib.set_prgname('lumen')
GLib.set_application_name('Lumen')
Gtk.Settings.get_default().set_property('gtk-application-prefer-dark-theme', True)
if icon and os.path.exists(icon):
    Gtk.Window.set_default_icon_from_file(icon)

win = Gtk.Window(title='Lumen')
win.set_default_size(1440, 900)
win.set_wmclass('lumen', 'lumen')

view = WebKit2.WebView()
color = Gdk.RGBA()
color.parse('#1a1b1d')
view.set_background_color(color)
settings = view.get_settings()
settings.set_property('enable-smooth-scrolling', True)
settings.set_property('enable-write-console-messages-to-stdout', False)

view.connect('notify::title', lambda v, _: win.set_title(v.get_title() or 'Lumen'))
view.connect('enter-fullscreen', lambda v: (win.fullscreen(), False)[1])
view.connect('leave-fullscreen', lambda v: (win.unfullscreen(), False)[1])


def unique(folder, name):
    base, ext = os.path.splitext(name)
    candidate = os.path.join(folder, name)
    n = 2
    while os.path.exists(candidate):
        candidate = os.path.join(folder, '%s (%d)%s' % (base, n, ext))
        n += 1
    return candidate


def on_destination(download, suggested):
    folder = GLib.get_user_special_dir(GLib.UserDirectory.DIRECTORY_DOWNLOAD) or os.path.expanduser('~')
    download.set_destination('file://' + unique(folder, suggested or 'download'))
    return True


WebKit2.WebContext.get_default().connect(
    'download-started', lambda ctx, d: d.connect('decide-destination', on_destination)
)

win.add(view)
win.connect('destroy', Gtk.main_quit)
win.show_all()
view.load_uri(url)
Gtk.main()
