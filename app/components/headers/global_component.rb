# typed: strict
# frozen_string_literal: true

module Headers
  class GlobalComponent < ApplicationComponent
    renders_one :breadcrumbs

    sig { params(signed_in: T::Boolean, current_page_name: PageName).void }
    def initialize(signed_in:, current_page_name:)
      @signed_in = signed_in
      @current_page_name = current_page_name
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name
  end
end
