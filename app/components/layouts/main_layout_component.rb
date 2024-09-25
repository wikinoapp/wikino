# typed: strict
# frozen_string_literal: true

module Layouts
  class MainLayoutComponent < ApplicationComponent
    sig { params(joined_lists: List::PrivateRelation, main_class_name: String).void }
    def initialize(joined_lists:, main_class_name: "")
      @joined_lists = joined_lists
      @main_class_name = main_class_name
    end

    sig { returns(List::PrivateRelation) }
    attr_reader :joined_lists
    private :joined_lists

    sig { returns(String) }
    attr_reader :main_class_name
    private :main_class_name
  end
end
