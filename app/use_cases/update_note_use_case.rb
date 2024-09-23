# typed: strict
# frozen_string_literal: true

class UpdateNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :note, Note
  end

  sig { params(viewer: User, note: Note, list: List, title: String, body: String).returns(Result) }
  def call(viewer:, note:, list:, title:, body:)
    note = ActiveRecord::Base.transaction do
      now = Time.zone.now

      note.attributes = {
        list:,
        title:,
        body:,
        body_html: Markup.new(text: body).render_html,
        modified_at: now
      }
      note.published_at = now if note.published_at.nil?
      note.save!

      note.add_editor!(editor: viewer)
      note.create_revision!(editor: viewer, body:, body_html: body)
      note.link!(editor: viewer)

      note
    end

    Result.new(note:)
  end
end
