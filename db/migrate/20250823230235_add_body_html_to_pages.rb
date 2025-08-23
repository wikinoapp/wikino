class AddBodyHtmlToPages < ActiveRecord::Migration[8.0]
  def change
    safety_assured do
      # pagesテーブルにbody_htmlカラムを追加
      add_column :pages, :body_html, :text unless column_exists?(:pages, :body_html)

      # page_revisionsテーブルにbody_htmlカラムを追加
      add_column :page_revisions, :body_html, :text unless column_exists?(:page_revisions, :body_html)

      # draft_pagesテーブルにbody_htmlカラムを追加
      add_column :draft_pages, :body_html, :text unless column_exists?(:draft_pages, :body_html)
    end
  end
end
