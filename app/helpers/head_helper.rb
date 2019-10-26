# frozen_string_literal: true

module HeadHelper
  def default_meta_tags
    site = "Nonoto"

    {
      reverse: true,
      site: site,
      separator: " |",
      description: meta_description,
      keywords: meta_keywords,
      og: {
        title: meta_tags.full_title(site: site, separator: " |"),
        type: "website",
        url: request.url,
        description: "A note app.",
        site_name: "Nonoto",
        image: "/og_image.png",
        locale: (I18n.locale == :ja ? "ja_JP" : "en_US")
      },
      twitter: {
        card: "summary",
        site: "",
        title: meta_tags.full_title(site: site, separator: " |"),
        description: "",
        image: "/og_image.png"
      },
      "turbolinks-cache-control": "no-cache"
    }
  end

  def meta_description(text = "")
    ary = []
    ary << "#{text} -" if text.present?
    ary << "A note app."
    ary.join(" ")
  end

  def meta_keywords(*keywords)
    default_keywords = "Nonoto".split(",")
    (keywords + default_keywords).join(",")
  end
end
