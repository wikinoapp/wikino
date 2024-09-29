# typed: strict
# frozen_string_literal: true

class UpdateNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :note, Note
  end

  sig { params(viewer: User, note: Note, topic: Topic, title: String, body: String).returns(Result) }
  def call(viewer:, note:, topic:, title:, body:)
    now = Time.zone.now

    note.attributes = {
      topic:,
      title:,
      body:,
      body_html: Markup.new(text: body).render_html,
      modified_at: now
    }
    note.published_at = now if note.published_at.nil?

    updated_note = ActiveRecord::Base.transaction do
      note.save!
      note.add_editor!(editor: viewer)
      note.create_revision!(editor: viewer, body:, body_html: body)
      note.link!(editor: viewer)
      viewer.destroy_draft_note!(note:)

      note
    end

    Result.new(note: updated_note)
  end
end
