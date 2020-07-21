# frozen_string_literal: true

module ApplicationHelper
  def site_theme
    cookies[:site_theme]
  end
end
