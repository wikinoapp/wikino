# typed: strict
# frozen_string_literal: true

module Headers
  class MainTitleComponent < ApplicationComponent
    renders_one :subtitle
    renders_one :actions

    sig { params(title: String, help_url: T.nilable(String)).void }
    def initialize(title:, help_url: nil)
      @title = title
      @help_url = help_url
    end

    sig { returns(String) }
    attr_reader :title
    private :title

    sig { returns(T.nilable(String)) }
    attr_reader :help_url
    private :help_url
  end
end
