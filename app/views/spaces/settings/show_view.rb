# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowView < ApplicationView
      use_helpers :set_meta_tags

      sig { params(space: Space, form: EditSpaceForm).void }
      def initialize(space:, form:)
        @space = space
        @form = form
      end

      sig { returns(Space) }
      attr_reader :space
      private :space

      sig { returns(EditSpaceForm) }
      attr_reader :form
      private :form

      sig { returns(PageName) }
      private def current_page_name
        PageName::SpaceSettings
      end
    end
  end
end
