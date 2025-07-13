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
      pages = if form.valid? && form.q.present?
        page_records = search_pages(form)
        PageRepository.new.to_models(page_records:)
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
        search_pages_with_space_filters(form)
      else
        keyword = form.q.not_nil!
        search_pages_all_user_spaces(keyword)
      end
    end

    # 指定されたスペース内のページを検索
    sig { params(form: Pages::SearchForm).returns(PageRecord::PrivateRelation) }
    private def search_pages_with_space_filters(form)
      space_identifiers = form.space_identifiers
      keyword = form.keyword_without_space_filters
      
      base_query = PageRecord
        .joins(:space_record)
        .joins("INNER JOIN space_members ON spaces.id = space_members.space_id")
        .where("space_members.user_id = ? AND space_members.active = ?", current_user_record!.id, true)
        .where("spaces.identifier IN (?)", space_identifiers)
        .active
        .order(modified_at: :desc)
        .limit(50)
      
      # キーワードが存在する場合のみタイトル検索を追加
      if keyword.present?
        base_query.where("pages.title ILIKE ?", "%#{keyword}%")
      else
        base_query
      end
    end

    # ユーザーが参加している全スペース内のページを検索
    sig { params(keyword: String).returns(PageRecord::PrivateRelation) }
    private def search_pages_all_user_spaces(keyword)
      PageRecord
        .joins(space_record: :space_member_records)
        .where(space_member_records: {user_id: current_user_record!.id})
        .where("pages.title ILIKE ?", "%#{keyword}%")
        .active
        .order(modified_at: :desc)
        .limit(50)
    end
  end
end
