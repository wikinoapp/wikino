# typed: strict
# frozen_string_literal: true

module Search
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      # 検索フォームの初期化
      form = Pages::SearchForm.new(
        q: params[:q].to_s.strip
      )

      # 検索結果の取得
      pages = if form.valid? && form.searchable?
        page_records = search_pages(form)
        PageRepository.new.to_models(page_records:, current_space_member: nil)
      else
        []
      end

      render_component Search::ShowView.new(
        form:,
        pages:,
        current_user: current_user!
      )
    end

    sig { params(form: Pages::SearchForm).returns(PageRecord::PrivateRelation) }
    private def search_pages(form)
      # space:フィルターが指定されている場合はそれを使用
      if form.has_space_filters?
        PageRecord.search_in_specific_spaces(
          user_record: current_user_record!,
          space_identifiers: form.space_identifiers,
          keywords: form.keywords_without_space_filters
        )
      else
        PageRecord.search_in_user_spaces(
          user_record: current_user_record!,
          keyword: form.q.not_nil!
        )
      end
    end
  end
end
