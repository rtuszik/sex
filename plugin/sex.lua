-- sex plugin for Rex, printed by `sex plugin --source`. SEX_VERSION and SEX_BIN are set by the
-- binary that printed it. Load it through the loader that `sex plugin` prints.
local M = { version = SEX_VERSION, bin = SEX_BIN }

local home = os.getenv("HOME") or ""
local extra_path = { "/opt/homebrew/bin", "/usr/local/bin", home .. "/go/bin", home .. "/.local/bin" }

local function exists(path)
  local f = io.open(path, "r")
  if f == nil then return false end
  f:close()
  return true
end

local function trim(s)
  return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

-- argv starts sex with the plugin's version, so sex can reload the config after an upgrade.
-- A package manager may already have removed the version this plugin came from.
local function argv(opts, ...)
  local bin = opts.bin
  if not exists(bin) and opts.fallback_bin ~= nil then bin = opts.fallback_bin end
  local out = { bin, "--plugin-version", M.version }
  for _, a in ipairs({ ... }) do out[#out + 1] = a end
  return out
end

local function picker_layout(opts, ctx, command)
  local cmd = argv(opts, unpack(command))
  if ctx.client_id ~= nil then
    cmd[#cmd + 1] = "--client"
    cmd[#cmd + 1] = ctx.client_id
  end
  if opts.fzf_args ~= nil and #opts.fzf_args > 0 then
    cmd[#cmd + 1] = "--"
    for _, a in ipairs(opts.fzf_args) do cmd[#cmd + 1] = a end
  end
  return rex.layout.block{
    flavor = "com.superlogical.terminal.shell",
    options = { command = cmd },
  }
end

local function open_picker(opts, ctx, command)
  local layout = picker_layout(opts, ctx, command)
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

-- opts.host: "layer" (default, a floating popup), "split", or "tab".
-- opts.bounds: popup { x, y, w, h } as window fractions.
-- opts.direction: "horizontal" or "vertical" for splits. opts.fzf_args: extra fzf options.
function M.setup(opts)
  opts = opts or {}
  opts.bin = M.bin
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
      local _, err = open_picker(opts, ctx, { "pick" })
      if err ~= nil then error(err, 0) end
    end,
  }

  rex.action{
    name = "sex_activity",
    title = "Pick Activity (sex)",
    category = "Sessions",
    keywords = "sex activity status agent blocked working waiting attention",
    run = function(ctx)
      local _, err = open_picker(opts, ctx, { "activity", "pick" })
      if err ~= nil then error(err, 0) end
    end,
  }

  -- Lets `sex doctor` see which plugin version is loaded, wherever the loader lives.
  rex.action{
    name = "sex_version",
    title = "Show Plugin Version (sex)",
    category = "Sessions",
    run = function()
      return { version = M.version, bin = opts.bin }
    end,
  }

  -- Asks sex for the previous session, then lets the app switch, so Remote Control is not needed.
  rex.action{
    name = "sex_last",
    title = "Last Session (sex)",
    category = "Sessions",
    keywords = "sex sesh session previous last back toggle",
    run = function(ctx)
      local cmd = argv(opts, "last", "--print")
      if ctx.client_id ~= nil then
        cmd[#cmd + 1] = "--client"
        cmd[#cmd + 1] = ctx.client_id
      end
      local r, err = rex.exec(cmd, { env = { PATH = path }, timeout = 10 })
      if r == nil then error(err, 0) end
      if not r.ok then error(trim(r.stderr), 0) end
      rex.client.queue("session.select", { session_id = trim(r.stdout) })
    end,
  }
end

return M
