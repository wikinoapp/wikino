# typed: strict
# frozen_string_literal: true

module Topics
  class TabsComponent < ApplicationComponent
    class TabItem < T::Struct
      const :label, String
      const :path, String
      const :active, T::Boolean
    end

    sig { params(tabs: T::Array[TabItem]).void }
    def initialize(tabs:)
      @tabs = tabs
    end

    sig { returns(T::Array[TabItem]) }
    attr_reader :tabs
    private :tabs
  end
end
