# typed: strict
# frozen_string_literal: true

module Search
  class ShowView < ApplicationView
    sig { params(form: Pages::SearchForm, pages: T::Array[Page], current_user: User).void }
    def initialize(form:, pages:, current_user:)
      @search_form = form
      @search_results = pages
      @current_user = current_user
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.search.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(Pages::SearchForm) }
    attr_reader :search_form
    private :search_form

    sig { returns(T::Array[Page]) }
    attr_reader :search_results
    private :search_results

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(PageName) }
    private def current_page_name
      PageName::Search
    end

    sig { returns(T::Boolean) }
    private def has_search_results?
      search_results.any?
    end

    sig { returns(T::Boolean) }
    private def show_no_results_message?
      search_form.searchable? && !has_search_results?
    end
  end
end