class MetricsBuffer implements Iterable<MetricsBuffer.Entry> {

    private final Iterable<Entry> mutable;
  
    MetricsBuffer(Iterable<MetricsBuffer.Entry> mutable) {
      this.mutable = mutable;
    }
  
    @Override
    public Iterator<Entry> iterator() {
      return mutable.iterator();
    }
  
    static class Entry {
      private final String sourceName;
      private final Iterable<MetricsRecordImpl> records;
  
      Entry(String name, Iterable<MetricsRecordImpl> records) {
        sourceName = name;
        this.records = records;
      }
  
      String name() {
        return sourceName;
      }
  
      Iterable<MetricsRecordImpl> records() {
        return records;
      }
    }
  
  }