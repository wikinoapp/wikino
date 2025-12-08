# typed: strict
# frozen_string_literal: true

module Topics
  module Shared
    class VisibilityFieldComponent < ApplicationComponent
      sig { params(form_builder: ActionView::Helpers::FormBuilder).void }
      def initialize(form_builder:)
        @form_builder = form_builder
      end

      sig { returns(ActionView::Helpers::FormBuilder) }
      attr_reader :form_builder
      private :form_builder
    end
  end
end
