# typed: strict
# frozen_string_literal: true

module CardLinks
  class PageComponent < ApplicationComponent
    sig { params(page: Page, show_topic_name: T::Boolean, card_class: String).void }
    def initialize(page:, show_topic_name: true, card_class: "")
      @page = page
      @show_topic_name = show_topic_name
      @card_class = card_class
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(T::Boolean) }
    attr_reader :show_topic_name
    private :show_topic_name
    alias_method :show_topic_name?, :show_topic_name

    sig { returns(String) }
    attr_reader :card_class
    private :card_class

    delegate :space, :topic, to: :page

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
