class RemoveBodyHtmlFromPages < ActiveRecord::Migration[8.0]
  def change
    safety_assured do
      remove_column :pages, :body_html, :text if column_exists?(:pages, :body_html)
      remove_column :page_revisions, :body_html, :text if column_exists?(:page_revisions, :body_html)
      remove_column :draft_pages, :body_html, :text if column_exists?(:draft_pages, :body_html)
    end
  end
end
