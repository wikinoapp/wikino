# frozen_string_literal: true

module ApplicationHelper
  def theme
    cookies[:theme]
  end
end
