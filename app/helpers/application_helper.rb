# frozen_string_literal: true

module ApplicationHelper
  def nonoto_config
    config = {
      locale: I18n.locale.to_s
    }

    javascript_tag "window.NonotoConfig = #{config.to_json.html_safe};"
  end
end
