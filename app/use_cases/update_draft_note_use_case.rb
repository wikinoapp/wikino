# typed: strict
# frozen_string_literal: true

class UpdateDraftNoteUseCase < ApplicationUseCase
  class Result < T::Struct
    const :draft_note, DraftNote
  end

  sig do
    params(
      viewer: User,
      note: Note,
      notebook_number: T.nilable(String),
      title: T.nilable(String),
      body: T.nilable(String)
    ).returns(Result)
  end
  def call(viewer:, note:, notebook_number:, title:, body:)
    updated_draft_note = ActiveRecord::Base.transaction do
      draft_note = viewer.find_or_create_draft_note!(note:)
      notebook = viewer.viewable_notebooks.find_by(number: notebook_number).presence || note.notebook
      new_body = body.presence || ""

      draft_note.attributes = {
        notebook:,
        title:,
        body: new_body,
        body_html: Markup.new(text: new_body).render_html,
        modified_at: Time.zone.now
      }
      draft_note.save!

      draft_note.link!(editor: viewer)

      draft_note
    end

    Result.new(draft_note: updated_draft_note)
  end
end
