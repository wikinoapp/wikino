# typed: strict
# frozen_string_literal: true

module Links
  class BrandIconComponent < ApplicationComponent
    sig { params(current_page_name: PageName, class_name: String).void }
    def initialize(current_page_name:, class_name: "")
      @current_page_name = current_page_name
      @class_name = class_name
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(T::Boolean) }
    def current_page?
      current_page_name.in?([PageName::Welcome, PageName::Home])
    end
  end
end
