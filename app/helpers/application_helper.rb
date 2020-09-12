# frozen_string_literal: true

module ApplicationHelper
  def color_scheme
    cookies[:color_scheme].presence || "system-default"
  end

  def page_title_with_suffix(page_title)
    "#{page_title} | Nonoto"
  end

  def nonoto_config
    config = {
      nonotoUrl: ENV.fetch("NONOTO_URL"),
      i18n: {
        messages: {
          createNoteWithKeyword: t('messages.edit_note.create_note_with_keyword')
        }
      }
    }.freeze

    javascript_tag "window.NonotoConfig = #{config.to_json.html_safe};"
  end
end
