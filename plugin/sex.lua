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

-- Rex clips a layer at its width and height measured from the window origin, not from the
-- layer's corner. Grow the layer by its offset and keep fzf out of the clipped margin.
local function layer_geometry(opts)
  local x, y, w, h = opts.bounds[1], opts.bounds[2], opts.bounds[3], opts.bounds[4]
  if not opts.clip_fix or (x == 0 and y == 0) then return { x, y, w, h }, nil end
  local lw, lh = math.min(x + w, 1 - x), math.min(y + h, 1 - y)
  local margin = string.format("--margin=0,%d%%,%d%%,0", math.ceil(x / lw * 100), math.ceil(y / lh * 100))
  return { x, y, lw, lh }, margin
end

local function picker_layout(opts, ctx, margin)
  local argv = { opts.bin, "pick" }
  if ctx.client_id ~= nil then
    argv[#argv + 1] = "--client"
    argv[#argv + 1] = ctx.client_id
  end
  local fzf_args = opts.fzf_args or {}
  if margin ~= nil or #fzf_args > 0 then
    argv[#argv + 1] = "--"
    if margin ~= nil then argv[#argv + 1] = margin end
    for _, a in ipairs(fzf_args) do argv[#argv + 1] = a end
  end
  return rex.layout.block{
    flavor = "com.superlogical.terminal.shell",
    options = { command = argv },
  }
end

local function open_picker(opts, ctx)
  if opts.host == "layer" then
    local bounds, margin = layer_geometry(opts)
    return rex.session.new_layer{ bounds = bounds, focus = true, layout = picker_layout(opts, ctx, margin) }
  end
  local layout = picker_layout(opts, ctx, nil)
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
-- opts.bounds: popup { x, y, w, h } as window fractions. opts.clip_fix: work around Rex's layer clipping (default true).
-- opts.direction: "horizontal" or "vertical" for splits. opts.fzf_args: extra fzf options.
function M.setup(opts)
  opts = opts or {}
  opts.bin = opts.bin or find_bin()
  opts.host = opts.host or "layer"
  opts.direction = opts.direction or "horizontal"
  opts.bounds = opts.bounds or { 0.1, 0.1, 0.8, 0.8 }
  if opts.clip_fix == nil then opts.clip_fix = true end
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
