# typed: strict
# frozen_string_literal: true

module ModelConcerns::NoteEditable
  extend ActiveSupport::Concern
  extend T::Sig

  sig { params(before: T.nilable(String), after: T.nilable(String), limit: Integer).returns(LinkList) }
  def fetch_link_list(before: nil, after: nil, limit: 15)
    added_note_ids = [id]

    page = linked_notes.where.not(id: added_note_ids).cursor_paginate(
      after:,
      before:,
      limit:,
      order: {modified_at: :desc, id: :desc}
    ).fetch
    notes = page.records
    page_info = PageInfo.from_cursor_paginate(page:)

    links = notes.map do |note|
      added_note_ids << note.id

      page = note.backlinked_notes.where.not(id: added_note_ids).cursor_paginate(
        after:,
        before:,
        limit:,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      backlinked_notes = page.records

      added_note_ids.concat(backlinked_notes.pluck(:id))

      Link.new(
        note:,
        backlinked_notes:,
        page_info: PageInfo.from_cursor_paginate(page:)
      )
    end

    LinkList.new(links:, page_info:)
  end
end
