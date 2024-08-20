# typed: strict
# frozen_string_literal: true

module Cards
  class ListCardComponent < ApplicationComponent
    sig { params(list: List, class_name: String).void }
    def initialize(list:, class_name: "")
      @list = list
      @class_name = class_name
    end

    sig { returns(List) }
    attr_reader :list
    private :list

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
