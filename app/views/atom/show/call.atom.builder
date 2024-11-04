atom_feed(root_url: space_url(Current.space!.identifier)) do |feed|
  feed.title("#{Current.space!.name} | Wikino")

  feed.updated(@pages[0].modified_at) if @pages.size > 0

  @pages.each do |page|
    feed.entry(page, url: false) do |entry|
      entry.link(href: page_url(Current.space!.identifier, page.number), rel: "alternate", type: "text/html")

      entry.title(page.title)
      entry.content(page.body, type: "text")

      feed.published(page.published_at)
      feed.updated(page.modified_at)
    end
  end
end
