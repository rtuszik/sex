-- One connection: every terminal that reported a program status, with its session and tab.
local terminal = "com.superlogical.terminal"
local list, err = rex.call("session.attach_all", { event_scope = "session" })
if err ~= nil then error(err, 0) end

local entries = {}
for _, s in ipairs(list.sessions or {}) do
  local windows = {}
  local view = rex.call("session.view", { session_id = s.session_id })
  if view ~= nil then
    for i, w in ipairs(view.windows or {}) do windows[w.window_id] = { label = w.label, index = i } end
  end
  local blocks = rex.call("session.list_blocks", { session_id = s.session_id })
  for _, b in ipairs((blocks or {}).blocks or {}) do
    if b.creator_name == terminal and b.window_id ~= nil then
      local args = { session_id = s.session_id, block_id = b.block_id, method = "program_status", args = {} }
      local status = rex.call("block.method", args)
      if status ~= nil and #(status.records or {}) > 0 then
        entries[#entries + 1] = {
          session_id = s.session_id,
          session = s.label,
          window_id = b.window_id,
          window = (windows[b.window_id] or {}).label,
          window_index = (windows[b.window_id] or {}).index,
          block_id = b.block_id,
          records = status.records,
        }
      end
    end
  end
end
return { entries = entries }
