# typed: strict
# frozen_string_literal: true

module Views
  module Atom
    class Show < Views::Base
      sig { params(props: Views::Atom::Show::Props).void }
      def initialize(props:)
        @props = props
      end

      sig { returns(Views::Atom::Show::Props) }
      attr_reader :props
      private :props

      delegate :space, :pages, to: :props

      def schema_date
        2025
      end

      def entry_id(page:)
        "tag:Wikino,#{schema_date}:Page/#{page.id}"
      end
    end
  end
end
