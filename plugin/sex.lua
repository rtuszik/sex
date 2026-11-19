-- sex plugin for Rex, printed by `sex plugin --source`. SEX_VERSION and SEX_BIN are set by the
-- binary that printed it. Load it through the loader that `sex plugin` prints.
local M = { version = SEX_VERSION, bin = SEX_BIN }

local home = os.getenv("HOME") or ""
local extra_path = { "/opt/homebrew/bin", "/usr/local/bin", home .. "/go/bin", home .. "/.local/bin" }

local function exists(path)
    local f = io.open(path, "r")
    if f == nil then
        return false
    end
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
    if not exists(bin) and opts.fallback_bin ~= nil then
        bin = opts.fallback_bin
    end
    local out = { bin, "--plugin-version", M.version }
    for _, a in ipairs({ ... }) do
        out[#out + 1] = a
    end
    return out
end

local source_icons = { rex = "terminal", config = "gear", zoxide = "folder" }

local function with_client(cmd, ctx)
    if ctx.client_id ~= nil then
        cmd[#cmd + 1] = "--client"
        cmd[#cmd + 1] = ctx.client_id
    end
    return cmd
end

local function run_sex(cmd, path)
    local r, err = rex.exec(cmd, { env = { PATH = path }, timeout = 10 })
    if r == nil then
        error(err, 0)
    end
    if not r.ok then
        error(trim(r.stderr), 0)
    end
    return r.stdout
end

local function decode(out)
    local v, err = rex.json.decode(out)
    if v == nil then
        error("sex: " .. tostring(err), 0)
    end
    return v
end

-- nil means cancelled; other failures raise, with a hint when the app predates prompts.
local function ask(spec)
    local choice, why, detail = rex.ui.pick(spec)
    if choice ~= nil or why == "cancelled" then
        return choice
    end
    if why == "unsupported" then
        error("sex: the Rex app cannot show palette prompts; update Rex", 0)
    end
    error("sex: " .. tostring(why) .. (detail and (": " .. detail) or ""), 0)
end

local function tilde(p)
    if p == nil or home == "" then
        return p
    end
    if p == home then
        return "~"
    end
    if p:sub(1, #home + 1) == home .. "/" then
        return "~" .. p:sub(#home + 1)
    end
    return p
end

-- The picker runs on the session's host, so sh finds sex there when this path is missing.
local function picker_layout(opts, ctx, label, command)
    local cmd = { "/bin/sh", "-c", SEX_RUN, "sex" }
    for _, a in ipairs(argv(opts, unpack(command))) do
        cmd[#cmd + 1] = a
    end
    with_client(cmd, ctx)
    if opts.fzf_args ~= nil and #opts.fzf_args > 0 then
        cmd[#cmd + 1] = "--"
        for _, a in ipairs(opts.fzf_args) do
            cmd[#cmd + 1] = a
        end
    end
    return rex.layout.block({
        flavor = "com.superlogical.terminal.shell",
        label = label,
        options = { command = cmd },
    })
end

local function open_picker(opts, ctx, label, command)
    local layout = picker_layout(opts, ctx, label, command)
    if opts.host == "layer" then
        return rex.session.new_layer({ bounds = opts.bounds, focus = true, layout = layout })
    end
    if opts.host == "split" and ctx.block_id ~= nil then
        return rex.session.new_split({
            anchor_id = ctx.block_id,
            direction = opts.direction,
            side = "after",
            focus = true,
            layout = layout,
        })
    end
    return rex.session.new_window({ layout = layout })
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

    rex.action({
        name = "sex_pick",
        title = "Pick Session (sex)",
        category = "Sessions",
        keywords = "sex sesh session switch project zoxide fzf",
        run = function(ctx)
            local _, err = open_picker(opts, ctx, "sex", { "pick" })
            if err ~= nil then
                error(err, 0)
            end
        end,
    })

    rex.action({
        name = "sex_activity",
        title = "Pick Activity (sex)",
        category = "Sessions",
        keywords = "sex activity status agent blocked working waiting attention",
        run = function(ctx)
            local _, err = open_picker(opts, ctx, "sex activity", { "activity", "pick" })
            if err ~= nil then
                error(err, 0)
            end
        end,
    })

    -- Palette pickers switch through the app, so Remote Control is not needed.
    rex.action({
        name = "sex_pick_palette",
        title = "Pick Session in Palette (sex)",
        category = "Sessions",
        keywords = "sex sesh session switch project zoxide palette",
        run = function(ctx)
            local list = decode(run_sex(with_client(argv(opts, "list", "--json", "--hide-current"), ctx), path))
            local items = {}
            for _, it in ipairs(list) do
                items[#items + 1] = {
                    id = it.session_id or it.name,
                    title = it.name,
                    subtitle = tilde(it.path),
                    icon = source_icons[it.source],
                    detail = it.source,
                }
            end
            local choice = ask({
                title = "Session",
                placeholder = "Filter sessions, or type a name or path",
                items = items,
                allow_other = true,
            })
            if choice == nil then
                return
            end
            if type(choice) == "table" then
                choice = choice.text
            end
            local id = trim(run_sex(argv(opts, "connect", "--background", "--print", "--", choice), path))
            rex.client.queue("session.select", { session_id = id })
        end,
    })

    rex.action({
        name = "sex_activity_palette",
        title = "Pick Activity in Palette (sex)",
        category = "Sessions",
        keywords = "sex activity status agent blocked working waiting attention palette",
        run = function()
            local list = decode(run_sex(argv(opts, "activity", "--json"), path))
            local items, byID = {}, {}
            for _, it in ipairs(list) do
                byID[it.block_id] = it
                local state = it.state
                if it.progress ~= nil then
                    state = state .. " " .. it.progress .. "%"
                end
                local sub = {}
                if it.app ~= nil then
                    sub[#sub + 1] = it.app
                end
                if it.message ~= nil then
                    sub[#sub + 1] = it.message
                end
                items[#items + 1] = {
                    id = it.block_id,
                    title = it.session .. " › " .. it.tab,
                    subtitle = #sub > 0 and table.concat(sub, "  ") or nil,
                    icon = "terminal",
                    detail = state,
                }
            end
            local choice = ask({ title = "Activity", placeholder = "Filter terminals", items = items })
            if choice == nil then
                return
            end
            local it = byID[choice]
            local _, err = rex.call("session.focus_block", { session_id = it.session_id, block_id = it.block_id })
            if err ~= nil then
                error(err, 0)
            end
            rex.client.queue("session.select", { session_id = it.session_id, window_id = it.window_id })
        end,
    })

    -- Lets `sex doctor` see which plugin version is loaded, wherever the loader lives.
    rex.action({
        name = "sex_version",
        title = "Show Plugin Version (sex)",
        category = "Sessions",
        run = function()
            return { version = M.version, bin = opts.bin }
        end,
    })

    -- Asks sex for the previous session, then lets the app switch, so Remote Control is not needed.
    rex.action({
        name = "sex_last",
        title = "Last Session (sex)",
        category = "Sessions",
        keywords = "sex sesh session previous last back toggle",
        run = function(ctx)
            local id = trim(run_sex(with_client(argv(opts, "last", "--print"), ctx), path))
            rex.client.queue("session.select", { session_id = id })
        end,
    })
end

return M
