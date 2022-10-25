- support olimpiici format

- Contests

Secret key pentru interactive (toata conversatia): https://discord.com/channels/@me/775486536358559764/956655627285962782

- Fix counting hidden submissions in BaseAPI.Submissions() limit
    - Maybe trim some data and still show the author?

- Automatically delete orphaned tests from subtasks in db, rn they are only hidden

- Note to self: when rewriting UI, look at the UI from kilonova.ro, it is better and more updated. the one in web/templ is pretty old and i don't have time to backport

UI:

- buton copiere exemplu

##########################

- PubSub with db/listener.go
  -> Use for submission update listening
  -> Websocket for submission scoring changes
- Use database for the data store
- moar websockets (ex: db stats in admin panel)
- upgrade codemirror to v6
- finish neater isolate binary access

Farther future:

- Use group perms for isolate binary (non-rooted grader, more secure, yay)
- Make web pretty to work with again
