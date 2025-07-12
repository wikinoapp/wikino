# typed: strict
# frozen_string_literal: true

module Headers
  class MainTitleComponent < ApplicationComponent
    renders_one :subtitle
    renders_one :actions

    sig { params(title: String).void }
    def initialize(title:)
      @title = title
    end

    sig { returns(String) }
    attr_reader :title
    private :title
  end
end
