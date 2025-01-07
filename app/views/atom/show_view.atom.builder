atom_feed(root_url: space_url(space.identifier), schema_date:) do |feed|
  feed.title("#{space.name} | Wikino")

  feed.updated(pages[0].modified_at) if pages.size > 0

  pages.each do |page|
    feed.entry(nil, {
      published: page.published_at,
      updated: page.modified_at,
      url: false,
      id: entry_id(page:)
    }) do |entry|
      entry.link(href: page_url(space.identifier, page.number), rel: "alternate", type: "text/html")

      entry.title(page.title)
      entry.content(page.body, type: "text")

      feed.published(page.published_at)
      feed.updated(page.modified_at)
    end
  end
end
