import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/Tabs';
import EventTab from './collection-center/EventTab';
import EvidenceTab from './collection-center/EvidenceTab';
import SourceTab from './collection-center/SourceTab';

export default function DataIngestionCenter({ token }: { token: string }) {
  return (
    <div className='grid h-full min-h-0 w-full grid-rows-[auto_minmax(0,1fr)] gap-3'>
      <div>
        <span className='page-eyebrow'>Data operations</span>
        <h2 className='page-title'>采集中心</h2>
        <p className='page-description'>查询标准化事件、完整证据与采集信源。</p>
      </div>
      <Tabs className='flex min-h-0 flex-col gap-3' defaultValue='events'>
        <TabsList aria-label='采集中心领域'>
          <TabsTrigger value='events'>事件中心</TabsTrigger>
          <TabsTrigger value='evidences'>证据中心</TabsTrigger>
          <TabsTrigger value='sources'>信源管理</TabsTrigger>
        </TabsList>
        <TabsContent className='min-h-0' value='events'>
          <EventTab token={token} />
        </TabsContent>
        <TabsContent className='min-h-0' value='evidences'>
          <EvidenceTab token={token} />
        </TabsContent>
        <TabsContent className='min-h-0' value='sources'>
          <SourceTab token={token} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
