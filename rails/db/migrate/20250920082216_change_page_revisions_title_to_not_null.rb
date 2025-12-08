class ChangePageRevisionsTitleToNotNull < ActiveRecord::Migration[8.0]
  def change
    safety_assured { change_column_null :page_revisions, :title, false }
  end
end
