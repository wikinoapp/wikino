# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView
    class HeaderComponent < ApplicationComponent
      sig { params(signed_in: T::Boolean, page_entity: PageEntity).void }
      def initialize(signed_in:, page_entity:)
        @signed_in = signed_in
        @page_entity = page_entity
      end

      sig { returns(T::Boolean) }
      attr_reader :signed_in
      private :signed_in
      alias_method :signed_in?, :signed_in

      sig { returns(PageEntity) }
      attr_reader :page_entity
      private :page_entity

      delegate :space_entity, to: :page_entity
    end
  end
end
