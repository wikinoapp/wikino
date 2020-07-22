# frozen_string_literal: true

module ApplicationHelper
  def site_theme
    cookies[:site_theme]
  end

  def page_title_with_suffix(page_title)
    "#{page_title} | Nonoto"
  end
end
