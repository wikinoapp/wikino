# typed: strict
# frozen_string_literal: true

module ModelConcerns::NoteEditable
  extend ActiveSupport::Concern
  extend T::Sig

  sig { params(before: T.nilable(String), after: T.nilable(String)).returns(T::Array[Link]) }
  def fetch_links(before: nil, after: nil)
    page = linked_notes.cursor_paginate(
      after:,
      before:,
      limit: 2,
      order: {modified_at: :desc, id: :desc}
    ).fetch
    notes = page.records
    page_info = PageInfo.from_cursor_paginate(page:)

    backlinked_note_data = notes.each_with_object({}) do |note, hash|
      page = note.backlinked_notes.cursor_paginate(
        after:,
        before:,
        limit: 2,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      backlinked_notes = page.records
      page_info = PageInfo.from_cursor_paginate(page:)

      hash[note.id] = {backlinked_notes:, page_info:}
    end

    added_note_ids = []

    links = notes.each do |note|
      added_note_ids << note.id

      # すでにリンクに追加されている記事は除外する
      backlinked_notes = backlinked_note_data[note.id][:backlinked_notes].filter do |backlinked_note|
        !backlinked_note.id.in?(added_note_ids)
      end

      added_note_ids.concat(backlinked_notes.map(&:id))

      Link.new(
        note:,
        backlinked_notes:,
        page_info: backlinked_note_data[note.id][:page_info]
      )
    end

    LinkList.new(links:, page_info:)
  end
end
