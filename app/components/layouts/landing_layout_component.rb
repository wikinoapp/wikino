# typed: strict
# frozen_string_literal: true

module Layouts
  class LandingLayoutComponent < ApplicationComponent
    sig { params(main_class_name: String).void }
    def initialize(main_class_name: "")
      @main_class_name = main_class_name
    end

    sig { returns(String) }
    attr_reader :main_class_name
    private :main_class_name
  end
end
