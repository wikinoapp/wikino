# typed: strict
# frozen_string_literal: true

module CardLinks
  class PageComponent < ApplicationComponent
    class CardImageSize < T::Enum
      enums do
        Small = new("small")
        Medium = new("medium")
      end
    end

    sig do
      params(
        page: Page,
        show_topic_name: T::Boolean,
        show_space_name: T::Boolean,
        options: T::Hash[Symbol, String],
        card_class: String,
        card_image_size: CardImageSize
      ).void
    end
    def initialize(
      page:,
      show_topic_name: true,
      show_space_name: false,
      options: {},
      card_class: "",
      card_image_size: CardImageSize::Medium
    )
      # `show_space_name: true & show_topic_name: false` の組み合わせは想定していない
      if show_space_name && !show_topic_name
        raise ArgumentError, "show_space_name: true requires show_topic_name: true"
      end

      @page = page
      @show_topic_name = show_topic_name
      @show_space_name = show_space_name
      @options = options
      @card_class = card_class
      @card_image_size = card_image_size
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

    sig { returns(CardImageSize) }
    attr_reader :card_image_size
    private :card_image_size

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

    sig { returns(String) }
    private def build_card_image_class
      size = card_image_size
      size_class = case size
      when CardImageSize::Small
        "h-16 w-16 md:h-12 md:w-12"
      when CardImageSize::Medium
        "h-16 w-16 md:h-18 md:w-18"
      else
        T.absurd(size)
      end

      class_names("rounded object-cover", size_class)
    end
  end
end
