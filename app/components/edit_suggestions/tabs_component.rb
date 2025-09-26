# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class TabsComponent < ApplicationComponent
    sig do
      params(
        page: Page,
        active_tab: Symbol
      ).void
    end
    def initialize(page:, active_tab:)
      @page = page
      @active_tab = active_tab
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(Symbol) }
    attr_reader :active_tab
    private :active_tab

    delegate :space, to: :page, private: true
    delegate :topic, to: :page, private: true

    sig { returns(T::Boolean) }
    private def create_new_active?
      active_tab == :create_new
    end

    sig { returns(T::Boolean) }
    private def add_to_existing_active?
      active_tab == :add_to_existing
    end

    sig { returns(String) }
    private def tab_link_classes
      "border-b-2 px-1 py-2 text-sm font-medium"
    end

    sig { returns(String) }
    private def tab_link_active_classes
      "border-indigo-500 text-indigo-600 dark:text-indigo-400"
    end

    sig { returns(String) }
    private def tab_link_inactive_classes
      "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
    end

    sig { returns(String) }
    private def create_new_link_classes
      class_names(
        tab_link_classes,
        tab_link_active_classes => create_new_active?,
        tab_link_inactive_classes => !create_new_active?
      )
    end

    sig { returns(String) }
    private def add_to_existing_link_classes
      class_names(
        tab_link_classes,
        tab_link_active_classes => add_to_existing_active?,
        tab_link_inactive_classes => !add_to_existing_active?
      )
    end
  end
end
