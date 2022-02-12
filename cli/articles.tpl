{{if eq .event "article_saved"}}💾 Article {{Foreground "#f54257" (print .article.ID)}} saved. (Tags: {{Foreground "#e3c1bf" (print .article.Tags)}})
{{else if eq .event "article_deleted"}}🚫 Article {{.article.ID}} deleted.
{{else if eq .event "article_show"}}🧑‍💻 {{Bold (Background "#0000ff" .article.Name) }}
----------------------------------------------
⌚️ {{ Italic "Created at"}} {{.article.CreatedAt}}
⌚️ {{ Italic "Last updated at"}} {{.article.UpdatedAt}}
{{if .article.Tags}}{{range .article.Tags}}🏷  {{Foreground "#e3c1bf" .}}
{{end}}{{end}}----------------------------------------------
{{if .article.Content}}{{.article.Content}}{{else}}<This article has no content>{{end}}
{{else if eq .event "article_list"}}{{if eq (len .articles) 0}}❗️ No articles found matching query.
{{else}}{{range .articles}}‣ {{Bold .Name}} {{Foreground "#f54257" (print "(" .ID ")")}} {{Foreground "#b5acad" (print .Tags)}}
{{end}}{{end}}{{end}}
