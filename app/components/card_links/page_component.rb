# typed: strict
# frozen_string_literal: true

module CardLinks
  class PageComponent < ApplicationComponent
    sig do
      params(
        page: Page,
        show_topic_name: T::Boolean,
        show_space_name: T::Boolean,
        options: T::Hash[Symbol, String],
        card_class: String
      ).void
    end
    def initialize(page:, show_topic_name: true, show_space_name: false, options: {}, card_class: "")
      # `show_space_name: true & show_topic_name: false` の組み合わせは想定していない
      if show_space_name && !show_topic_name
        raise ArgumentError, "show_space_name: true requires show_topic_name: true"
      end

      @page = page
      @show_topic_name = show_topic_name
      @show_space_name = show_space_name
      @options = options
      @card_class = card_class
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(T::Boolean) }
    attr_reader :show_topic_name
    private :show_topic_name
    alias_method :show_topic_name?, :show_topic_name

    sig { returns(T::Boolean) }
    attr_reader :show_space_name
    private :show_space_name
    alias_method :show_space_name?, :show_space_name

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { returns(String) }
    attr_reader :card_class
    private :card_class

    delegate :space, :topic, to: :page

    sig { returns(String) }
    private def build_class_name
      class_names(options[:class])
    end

    sig { returns(T::Hash[Symbol, String]) }
    private def build_options
      options.merge(
        class: build_class_name
      )
    end

    sig { returns(String) }
    private def build_card_class
      class_names(
        card_class,
        "bg-card duration-200 ease-in-out grid min-h-[96px] transition px-3 py-2",
        "hover:border hover:border-primary",
        relative: page.pinned?
      )
    end
  end
end
