-- One connection: every session with its visible block, cwd, and when the app last showed it.
local client = rex.args.client
local list, err = rex.call("session.attach_all", { event_scope = "session" })
if err ~= nil then error(err, 0) end

local sessions = {}
for _, s in ipairs(list.sessions or {}) do
  local item = { session_id = s.session_id, label = s.label }
  local view = rex.call("session.view", { session_id = s.session_id })
  if view ~= nil then
    for i, w in ipairs(view.windows or {}) do
      if i == 1 then item.first_window_id = w.window_id end
      if w.window_id == view.active_window_id then item.block_id = w.focused_block_id end
    end
  end
  if item.block_id ~= nil then
    local args = { session_id = s.session_id, block_id = item.block_id, args = {} }
    args.method = "size"
    local size = rex.call("block.method", args)
    if size ~= nil and client ~= nil then
      for _, d in ipairs(size.desired or {}) do
        if d.client_id == client then item.shown_at = d.updated_at end
      end
    end
    args.method = "process"
    local proc = rex.call("block.method", args)
    if proc ~= nil and proc.foreground ~= nil then item.cwd = proc.foreground.cwd end
  end
  sessions[#sessions + 1] = item
end
return { sessions = sessions }
