package bundles

// The shims define the $dz / dz API exactly like Dropzone 4's scripting
// interface and translate calls into a line protocol on stdout that the Go
// runner parses (see action.go). Items arrive as argv, options as env vars.

const rubyShim = `# frozen_string_literal: true
STDOUT.sync = true

class DropzoneAPI
  def begin(msg) emit("BEGIN", msg) end
  def determinate(v) emit("DETERMINATE", v ? "true" : "false") end
  def percent(p) emit("PERCENT", p.to_i.to_s) end
  def finish(msg) emit("FINISH", msg) end
  def url(u, title = nil) emit("URL", u == false ? "" : u.to_s) end
  def text(t) emit("TEXT", t.to_s) end
  def fail(msg) emit("FAIL", msg); exit(1) end
  def error(title, msg) emit("ERROR", "#{title}#{msg}"); exit(1) end
  def alert(title, msg) emit("ALERT", "#{title}#{msg}") end
  def save_value(name, value) emit("SAVE", "#{name}#{value}") end
  def remove_value(name) emit("REMOVE", name) end
  def read_clipboard() %x(pbpaste) end
  def temp_folder() ENV["DZ_TEMP"] end
  def add_dropbar(items) Array(items).each { |i| emit("DROPBAR", i.to_s) } end
  def inputbox(title, prompt, field = nil)
    emit("INPUTBOX", "#{title}#{prompt}")
    reply = STDIN.gets
    fail("No input provided") if reply.nil? || reply.strip.empty?
    reply.strip.gsub("", "\n")
  end
  def pashua(config) fail("pashua dialogs are not supported by this action host yet") end

  private

  def emit(kind, payload)
    puts "DZX:#{kind}:#{payload.to_s.gsub("\n", "")}"
  end
end

$dz = DropzoneAPI.new
$items = ARGV

require ENV["DZ_ACTION_SCRIPT"]

if ENV["DZ_EVENT"] == "clicked"
  clicked
else
  dragged
end
`

const pythonShim = `import os
import subprocess
import sys

class DropzoneAPI:
    def _emit(self, kind, payload):
        sys.stdout.write("DZX:%s:%s\n" % (kind, str(payload).replace("\n", "")))
        sys.stdout.flush()

    def begin(self, msg): self._emit("BEGIN", msg)
    def determinate(self, v): self._emit("DETERMINATE", "true" if v else "false")
    def percent(self, p): self._emit("PERCENT", int(p))
    def finish(self, msg): self._emit("FINISH", msg)
    def url(self, u, title=None): self._emit("URL", "" if u is False else u)
    def text(self, t): self._emit("TEXT", t)
    def fail(self, msg):
        self._emit("FAIL", msg)
        sys.exit(1)
    def error(self, title, msg):
        self._emit("ERROR", "%s%s" % (title, msg))
        sys.exit(1)
    def alert(self, title, msg): self._emit("ALERT", "%s%s" % (title, msg))
    def save_value(self, name, value): self._emit("SAVE", "%s%s" % (name, value))
    def remove_value(self, name): self._emit("REMOVE", name)
    def read_clipboard(self): return subprocess.check_output(["pbpaste"]).decode("utf-8")
    def temp_folder(self): return os.environ.get("DZ_TEMP", "/tmp")
    def add_dropbar(self, items):
        for i in items if isinstance(items, (list, tuple)) else [items]:
            self._emit("DROPBAR", i)
    def inputbox(self, title, prompt, field=None):
        self._emit("INPUTBOX", "%s%s" % (title, prompt))
        reply = sys.stdin.readline()
        if not reply or not reply.strip():
            self.fail("No input provided")
        return reply.strip().replace("", "\n")
    def pashua(self, config): self.fail("pashua dialogs are not supported by this action host yet")

dz = DropzoneAPI()
items = sys.argv[1:]

script_path = os.environ["DZ_ACTION_SCRIPT"]
namespace = {"dz": dz, "items": items, "__name__": "__main__"}
with open(script_path) as f:
    code = compile(f.read(), script_path, "exec")
exec(code, namespace)

event = os.environ.get("DZ_EVENT", "dragged")
handler = namespace.get("clicked" if event == "clicked" else "dragged")
if handler is None:
    dz.fail("action does not define %s()" % event)
handler()
`
