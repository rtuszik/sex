-- sex plugin for Rex. Save next to init.lua (`sex plugin > ~/.config/rex/sex.lua`), then in init.lua:
--   require("sex").setup()
--   rex.bind("ctrl+b>shift+t", "sex_pick")
--   rex.bind("ctrl+b>shift+l", "sex_last")
local M = {}

local home = os.getenv("HOME") or ""
local extra_path = {
  "/opt/homebrew/bin",
  "/usr/local/bin",
  home .. "/go/bin",
  home .. "/.local/bin",
  home .. "/.local/share/mise/shims",
}

local function exists(path)
  local f = io.open(path, "r")
  if f == nil then return false end
  f:close()
  return true
end

local function find_bin()
  for _, dir in ipairs(extra_path) do
    if exists(dir .. "/sex") then return dir .. "/sex" end
  end
  return "sex"
end

local function trim(s)
  return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

local function picker_layout(opts, ctx)
  local argv = { opts.bin, "pick" }
  if ctx.client_id ~= nil then
    argv[#argv + 1] = "--client"
    argv[#argv + 1] = ctx.client_id
  end
  if opts.fzf_args ~= nil and #opts.fzf_args > 0 then
    argv[#argv + 1] = "--"
    for _, a in ipairs(opts.fzf_args) do argv[#argv + 1] = a end
  end
  return rex.layout.block{
    flavor = "com.superlogical.terminal.shell",
    options = { command = argv },
  }
end

local function open_picker(opts, ctx)
  local layout = picker_layout(opts, ctx)
  if opts.host == "layer" then
    return rex.session.new_layer{ bounds = opts.bounds, focus = true, layout = layout }
  end
  if opts.host == "split" and ctx.block_id ~= nil then
    return rex.session.new_split{
      anchor_id = ctx.block_id,
      direction = opts.direction,
      side = "after",
      focus = true,
      layout = layout,
    }
  end
  return rex.session.new_window{ layout = layout }
end

-- opts.bin: path to sex. opts.host: "layer" (default, a floating popup), "split", or "tab".
-- opts.bounds: popup { x, y, w, h } as window fractions.
-- opts.direction: "horizontal" or "vertical" for splits. opts.fzf_args: extra fzf options.
function M.setup(opts)
  opts = opts or {}
  opts.bin = opts.bin or find_bin()
  opts.host = opts.host or "layer"
  opts.direction = opts.direction or "horizontal"
  opts.bounds = opts.bounds or { 0.1, 0.1, 0.8, 0.8 }
  local path = table.concat(extra_path, ":") .. ":" .. (os.getenv("PATH") or "/usr/bin:/bin")

  rex.action{
    name = "sex_pick",
    title = "Pick Session (sex)",
    category = "Sessions",
    keywords = "sex sesh session switch project zoxide fzf",
    run = function(ctx)
      local _, err = open_picker(opts, ctx)
      if err ~= nil then error(err, 0) end
    end,
  }

  -- Asks sex for the previous session, then lets the app switch, so Remote Control is not needed.
  rex.action{
    name = "sex_last",
    title = "Last Session (sex)",
    category = "Sessions",
    keywords = "sex sesh session previous last back toggle",
    run = function(ctx)
      local argv = { opts.bin, "last", "--print" }
      if ctx.client_id ~= nil then
        argv[#argv + 1] = "--client"
        argv[#argv + 1] = ctx.client_id
      end
      local r, err = rex.exec(argv, { env = { PATH = path }, timeout = 10 })
      if r == nil then error(err, 0) end
      if not r.ok then error(trim(r.stderr), 0) end
      rex.client.queue("session.select", { session_id = trim(r.stdout) })
    end,
  }
end

return M
