# frozen_string_literal: true

require "github/markup"

module Mutations
  class UpdateNote < Mutations::Base
    argument :id, ID, required: true
    argument :body, String, required: true

    field :note, Types::Objects::NoteType, null: true
    field :error, Types::Objects::UpdateNoteErrorType, null: true

    def resolve(id:, body:)
      note = NonotoSchema.object_from_id(id, context)
      note.body = body
      note.body_html = render_html(body)
      note.set_title!
      note.modified_at = Time.zone.now

      unless note.valid?
        error_detail = note.errors.details.dig(:title, 0, :error)

        if error_detail == :taken
          viewer = context[:viewer]
          original_note = viewer.notes.where.not(id: note.id).find_by(title: note.title)

          return {
            note: original_note,
            error: {
              code: "DUPLICATED"
            }
          }
        end

        return {
          note: nil,
          error: {
            code: "INVALID"
          }
        }
      end

      ActiveRecord::Base.transaction(joinable: false, requires_new: true) do
        note.save!
        note.link!
      end

      {
        note: note,
        error: nil
      }
    end

    private

    def render_html(body)
      GitHub::Markup.render_s(
        GitHub::Markups::MARKUP_MARKDOWN,
        body,
        options: { commonmarker_opts: %i(HARDBREAKS) }
      )
    end
  end
end
