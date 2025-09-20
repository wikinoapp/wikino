class AddTitleToPageRevisions < ActiveRecord::Migration[8.0]
  def change
    add_column :page_revisions, :title, :citext, null: true
  end
end
