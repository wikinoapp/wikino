# typed: strict
# frozen_string_literal: true

module Views
  module Atom
    class Show
      class Props < T::Struct
        class Space < T::Struct
          const :identifier, String
          const :name, String
        end

        class Page < T::Struct
          const :id, T::Wikino::DatabaseId
          const :number, Integer
          const :title, String
          const :body, String
          const :published_at, ActiveSupport::TimeWithZone
          const :modified_at, ActiveSupport::TimeWithZone
        end

        const :space, Space
        const :pages, T::Array[Page]

        def self.build(space:, pages:)
          new(
            space: Space.new(
              identifier: space.identifier,
              name: space.name
            ),
            pages: pages.map do |page|
              Page.new(
                id: page.id,
                number: page.number,
                title: page.title,
                body: page.body,
                modified_at: page.modified_at,
                published_at: page.published_at
              )
            end
          )
        end
      end
    end
  end
end
