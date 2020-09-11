# frozen_string_literal: true

require "github/markup"

module Mutations
  class UpdateNote < Mutations::Base
    argument :id, ID, required: true
    argument :body, String, required: true

    field :note, Types::Objects::NoteType, null: false
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(id:, body:)
      note = NonotoSchema.object_from_id(id, context)
      note.body = body
      note.body_html = render_html(body)
      note.set_title!

      unless note.valid?
        return {
          note: nil,
          errors: note.errors.full_messages.map { |msg| { message: msg } }
        }
      end

      note.save

      {
        note: note,
        errors: []
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
