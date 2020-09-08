# frozen_string_literal: true

require "github/markup"

module Types
  module Objects
    class NoteType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      global_id_field :id

      field :database_id, String, null: false
      field :title, String, null: false
      field :body, String, null: false
      field :body_html, String, null: false
      field :updated_at, GraphQL::Types::ISO8601DateTime, null: false

      def body_html
        GitHub::Markup.render_s(
          GitHub::Markups::MARKUP_MARKDOWN,
          object.body,
          options: { commonmarker_opts: %i(HARDBREAKS) }
        )
      end
    end
  end
end
