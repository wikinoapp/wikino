# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView
    class HeaderComponent < ApplicationComponent
      sig { params(signed_in: T::Boolean, page: Page).void }
      def initialize(signed_in:, page:)
        @signed_in = signed_in
        @page = page
      end

      sig { returns(T::Boolean) }
      attr_reader :signed_in
      private :signed_in
      alias_method :signed_in?, :signed_in

      sig { returns(Page) }
      attr_reader :page
      private :page

      delegate :space, to: :page
    end
  end
end
