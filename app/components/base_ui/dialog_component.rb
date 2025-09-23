# typed: strict
# frozen_string_literal: true

module BaseUI
  class DialogComponent < ApplicationComponent
    sig { params(options: T::Hash[Symbol, String]).void }
    def initialize(options:)
      @options = options
      build_dialog_classes
    end

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { void }
    def build_dialog_classes
      @options[:class] = [
        "dialog",
        "w-full sm:max-w-2xl",
        "rounded-lg",
        "bg-white",
        "p-0",
        "shadow-xl",
        "backdrop:bg-black/50",
        @options[:class]
      ].compact.join(" ")
    end
  end
end
