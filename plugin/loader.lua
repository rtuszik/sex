-- sex loader for Rex, printed by `sex plugin`. Save it where require finds it, for
-- example as sex.lua next to init.lua, then call require("sex").setup({ ... }) in init.lua.
-- On each config load it asks the installed sex for the plugin, so upgrades need no new copy.
local home = os.getenv("HOME") or ""
local user = os.getenv("USER") or ""
local candidates = {
  os.getenv("SEX_BIN"),
  "/opt/homebrew/bin/sex",
  "/usr/local/bin/sex",
  home .. "/go/bin/sex",
  home .. "/.local/bin/sex",
  home .. "/.nix-profile/bin/sex",
  "/etc/profiles/per-user/" .. user .. "/bin/sex",
  "/run/current-system/sw/bin/sex",
  home .. "/.local/share/mise/shims/sex",
  home .. "/.asdf/shims/sex",
  "/usr/bin/sex",
}

local function exists(path)
  local f = io.open(path, "r")
  if f == nil then return false end
  f:close()
  return true
end

local function find()
  for _, p in ipairs(candidates) do
    if p ~= nil and p ~= "" and exists(p) then return p end
  end
  return "sex"
end

local M = {}

-- opts.bin overrides the search. All other options go to the plugin.
function M.setup(opts)
  opts = opts or {}
  local found = opts.bin or find()
  local r, err = rex.exec({ found, "plugin", "--source" }, { timeout = 10 })
  if r == nil or not r.ok then
    rex.log("warn", "sex: could not load the plugin from " .. found .. ": " .. tostring(err or r.stderr))
    return
  end
  local chunk, lerr = loadstring(r.stdout, "=sex plugin")
  if chunk == nil then
    rex.log("warn", "sex: " .. lerr)
    return
  end
  opts.fallback_bin = found
  chunk().setup(opts)
end

return M
