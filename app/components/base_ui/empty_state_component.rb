# typed: strict
# frozen_string_literal: true

module BaseUI
  class EmptyStateComponent < ApplicationComponent
    sig { params(icon_name: String, message: String, options: T::Hash[Symbol, String]).void }
    def initialize(icon_name:, message:, options: {})
      @icon_name = icon_name
      @message = message
      @options = options
    end

    sig { returns(String) }
    attr_reader :icon_name
    private :icon_name

    sig { returns(String) }
    attr_reader :message
    private :message

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { returns(String) }
    private def build_class_name
      class_names("flex flex-col gap-4 text-center", options[:class])
    end

    sig { returns(T::Hash[Symbol, String]) }
    private def build_options
      options.merge(
        class: build_class_name
      )
    end
  end
end
